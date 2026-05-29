package jira

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const maxWebhookBodySize = 5 * 1024 * 1024 // 5MB

// WebhookHandler receives and routes Jira webhook events.
// Registered at POST /webhooks/jira
//
// Jira webhook authentication: the secret is validated via a constant-time
// comparison of the ?secret= query parameter. Jira does not support
// HMAC-SHA256 signatures on webhook payloads.
type WebhookHandler struct {
	secret      string
	router      *EventRouter
	responder   interface{} // response.Responder (stub)
	installRepo InstallationRepository
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(secret string, router *EventRouter, responder interface{}, installRepo InstallationRepository) *WebhookHandler {
	return &WebhookHandler{
		secret:      secret,
		router:      router,
		responder:   responder,
		installRepo: installRepo,
	}
}

// Handle is the HTTP handler for POST /webhooks/jira?secret=<value>&cloud_id=<value>
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Step 1: Validate the secret parameter — constant-time comparison
	incomingSecret := r.URL.Query().Get("secret")
	if !h.validateSecret(incomingSecret) {
		slog.Warn("jira webhook: invalid secret",
			"cloud_id", r.URL.Query().Get("cloud_id"),
			"remote_addr", r.RemoteAddr,
		)
		// TODO: use responder.Error
		http.Error(w, "invalid webhook secret", http.StatusUnauthorized)
		return
	}

	cloudID := r.URL.Query().Get("cloud_id")
	if cloudID == "" {
		// TODO: use responder.Error
		http.Error(w, "missing cloud_id parameter", http.StatusBadRequest)
		return
	}

	// Step 2: Resolve the installation from the cloudId
	installation, err := h.installRepo.FindByCloudID(ctx, cloudID)
	if err != nil {
		slog.Warn("jira webhook: unknown cloud_id", "cloud_id", cloudID)
		// Return 200 to prevent Jira from retrying for unknown sites
		w.WriteHeader(http.StatusOK)
		return
	}

	// Step 3: Read and parse the payload
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize))
	if err != nil {
		// TODO: use responder.Error
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}

	// Step 4: Acknowledge immediately
	w.WriteHeader(http.StatusOK)

	// Step 5: Route asynchronously
	go func() {
		routeCtx := context.Background()
		if err := h.router.Route(routeCtx, installation.OrgID, body); err != nil {
			slog.Error("jira webhook: routing failed",
				"cloud_id", cloudID,
				"org_id", installation.OrgID,
				"error", err,
			)
		}
	}()
}

// validateSecret performs a constant-time comparison of the webhook secret.
// Never log the incoming or expected secret value.
func (h *WebhookHandler) validateSecret(incoming string) bool {
	if incoming == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(incoming), []byte(h.secret)) == 1
}

// EventRouter dispatches Jira webhook events to the appropriate processor.
type EventRouter struct {
	linkageService interface{} // LinkageService (stub)
	queue          interface{} // JobQueue (stub)
}

// NewEventRouter creates a new EventRouter.
func NewEventRouter(linkageService interface{}, queue interface{}) *EventRouter {
	return &EventRouter{
		linkageService: linkageService,
		queue:          queue,
	}
}

// LinkageService is the interface the linkage domain exposes to this package.
type LinkageService interface {
	UpdateFromJiraEvent(ctx context.Context, event *JiraWebhookEvent) error
}

// JiraWebhookEvent is the parsed representation of a Jira webhook payload.
type JiraWebhookEvent struct {
	WebhookEvent string     `json:"webhookEvent"` // e.g. "jira:issue_updated"
	Issue        *JiraIssue `json:"issue"`
	ChangeLog    *ChangeLog `json:"changelog"`
	OrgID        uint64     // set by the webhook handler from installation lookup
}

type ChangeLog struct {
	Items []ChangeLogItem `json:"items"`
}

type ChangeLogItem struct {
	Field      string `json:"field"`
	FieldType  string `json:"fieldType"`
	From       string `json:"from"`
	FromString string `json:"fromString"`
	To         string `json:"to"`
	ToString   string `json:"toString"`
}

func (r *EventRouter) Route(ctx context.Context, orgID uint64, body []byte) error {
	var event JiraWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("parse jira webhook: %w", err)
	}
	event.OrgID = orgID

	switch event.WebhookEvent {
	case "jira:issue_updated":
		// An issue's status changed — update any linked DeployEvent records.
		// Most relevant when a Jira issue transitions to "Done" or "Approved"
		// after the deploy was already linked.
		// TODO: use linkageService.UpdateFromJiraEvent
		return nil
	case "jira:issue_deleted":
		// Soft-mark the linked ticket as deleted — do not remove the link.
		// The evidence record must remain even if the Jira issue is deleted.
		slog.Info("jira: issue deleted — preserving deploy linkage for audit trail",
			"issue_key", event.Issue.Key,
		)
		// TODO: use linkageService.UpdateFromJiraEvent
		return nil
	default:
		slog.Debug("jira webhook: unhandled event type", "event", event.WebhookEvent)
		return nil
	}
}

// RegisterWebhook registers a Jira webhook for issue events on the customer's site.
// Called once during installation setup. The webhook URL includes the secret
// and cloudId as query parameters — Jira will POST to this URL for each event.
//
// Registered events: jira:issue_updated, jira:issue_deleted
func RegisterWebhook(ctx context.Context, client *Client, secret, webhookBaseURL string) (int, error) {
	webhookURL := fmt.Sprintf(
		"%s/webhooks/jira?secret=%s&cloud_id=%s",
		webhookBaseURL, secret, client.cloudID,
	)

	payload := map[string]any{
		"name": "DeployGuard Deploy Linkage",
		"url":  webhookURL,
		"events": []string{
			"jira:issue_updated",
			"jira:issue_deleted",
		},
		"filters": map[string]string{
			// Only trigger on issues relevant to deployment change tracking.
			// Customers can configure their project key prefix in Gate 2.
			"issue-related-events-section": "",
		},
		"excludeBody": false,
	}

	data, _ := json.Marshal(payload)
	resp, err := client.Do(ctx, http.MethodPost,
		"/rest/webhooks/1.0/webhook",
		bytes.NewReader(data),
	)
	if err != nil {
		return 0, fmt.Errorf("register jira webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("register jira webhook failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Self string `json:"self"`
		ID   int    `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.ID, nil
}
