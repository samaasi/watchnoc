package pagerduty

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const maxWebhookBodySize = 5 * 1024 * 1024

// WebhookHandler receives and routes PagerDuty webhook events.
type WebhookHandler struct {
	webhookSecret string
	installRepo   InstallationRepository
	correlator    *Correlator
	responder     interface{}
}

// InstallationRepository is the interface for storing and retrieving PagerDuty installations.
type InstallationRepository interface {
	FindByOrgID(ctx context.Context, orgID uint64) (*Installation, error)
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(secret string, installRepo InstallationRepository, correlator *Correlator, responder interface{}) *WebhookHandler {
	return &WebhookHandler{
		webhookSecret: secret,
		installRepo:   installRepo,
		correlator:    correlator,
		responder:     responder,
	}
}

// Handle is the HTTP handler for PagerDuty Generic V3 Webhooks.
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize))
	if err != nil {
		slog.Error("failed to read webhook body", "error", err)
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}

	if !h.validateSignature(r, body) {
		slog.Warn("pagerduty webhook: invalid signature", "remote_addr", r.RemoteAddr)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload V3WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("failed to parse webhook payload", "error", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Acknowledge the webhook immediately
	w.WriteHeader(http.StatusOK)

	// Process asynchronously
	go func() {
		incident := h.normalizeIncident(&payload)
		switch payload.Event {
		case "incident.triggered":
			if err := h.correlator.LinkIncidentToDeploy(ctx, incident); err != nil {
				slog.Error("failed to link incident to deploy", "error", err, "incident_id", incident.ID)
			}
		case "incident.resolved":
			if err := h.correlator.HandleResolvedIncident(ctx, incident); err != nil {
				slog.Error("failed to handle resolved incident", "error", err, "incident_id", incident.ID)
			}
		}
	}()
}

// validateSignature validates the HMAC-SHA256 signature from PagerDuty.
func (h *WebhookHandler) validateSignature(r *http.Request, body []byte) bool {
	signatures := r.Header.Get("X-PagerDuty-Signature")
	if signatures == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedSignature := "v1=" + hex.EncodeToString(mac.Sum(nil))

	for _, sig := range strings.Split(signatures, ",") {
		if subtle.ConstantTimeCompare([]byte(sig), []byte(expectedSignature)) == 1 {
			return true
		}
	}

	return false
}

func (h *WebhookHandler) normalizeIncident(payload *V3WebhookPayload) *PagerDutyIncident {
	return &PagerDutyIncident{
		ID:         payload.Data.ID,
		Title:      payload.Data.Title,
		Status:     payload.Data.Status,
		Urgency:    payload.Data.Urgency,
		HTMLURL:    payload.Data.HTMLURL,
		CreatedAt:  payload.Data.CreatedAt,
		ResolvedAt: payload.Data.ResolvedAt,
		Service: PagerDutyService{
			ID:   payload.Data.Service.ID,
			Name: payload.Data.Service.Name,
			URL:  payload.Data.Service.Self,
		},
	}
}
