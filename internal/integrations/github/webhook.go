// internal/integrations/github/webhook.go

package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	gogithub "github.com/google/go-github/v62/github"
)

const (
	// maxWebhookBodySize prevents memory exhaustion from malformed payloads.
	// GitHub webhook payloads are typically under 100KB.
	// We allow up to 10MB to handle edge cases with large diffs.
	maxWebhookBodySize = 10 * 1024 * 1024 // 10MB

	// signatureHeader is the header GitHub sets on every webhook request.
	signatureHeader = "X-Hub-Signature-256"

	// deliveryHeader is GitHub's unique ID for each webhook delivery.
	// Used for idempotency — if GitHub retries, same delivery ID arrives.
	deliveryHeader = "X-GitHub-Delivery"

	// eventHeader identifies the event type without parsing the body.
	eventHeader = "X-GitHub-Event"
)

// WebhookHandler receives, validates, and routes GitHub webhook events.
type WebhookHandler struct {
	secret      string
	router      *EventRouter
	responder   interface{} // response.Responder
	installRepo InstallationRepository
}

func NewWebhookHandler(
	secret string,
	router *EventRouter,
	responder interface{},
	installRepo InstallationRepository,
) *WebhookHandler {
	return &WebhookHandler{
		secret:      secret,
		router:      router,
		responder:   responder,
		installRepo: installRepo,
	}
}

// Handle is the HTTP handler registered at POST /webhooks/github.
// Every GitHub webhook delivery for the App arrives here.
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Step 1: Read body with size limit
	// Body must be read BEFORE validation — the signature covers the raw bytes
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize))
	if err != nil {
		slog.Warn("github webhook: failed to read body", "error", err)
		// TODO: use responder
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Step 2: Validate HMAC-SHA256 signature
	// This MUST happen before any payload parsing — reject unsigned requests immediately
	signature := r.Header.Get(signatureHeader)
	if !h.validateSignature(body, signature) {
		slog.Warn("github webhook: invalid signature",
			"delivery_id", r.Header.Get(deliveryHeader),
			"remote_addr", r.RemoteAddr,
		)
		// Return 401 — do not reveal why validation failed
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Step 3: Extract metadata headers
	eventType := r.Header.Get(eventHeader)
	deliveryID := r.Header.Get(deliveryHeader)

	if eventType == "" || deliveryID == "" {
		// TODO: use responder
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slog.Info("github webhook received",
		"event_type", eventType,
		"delivery_id", deliveryID,
		"body_size", len(body),
	)

	// Step 4: Acknowledge immediately — GitHub expects a response within 10 seconds.
	// All processing is async via Asynq. We respond 202 now, process later.
	// If we return a non-2xx, GitHub will retry (which is correct behaviour on failure).
	w.WriteHeader(http.StatusAccepted)

	// Step 5: Route to the appropriate handler asynchronously
	// The raw body is passed so workers can store it for replay/debugging
	go func() {
		// Use a fresh context — the request context is cancelled when
		// the HTTP response is sent
		routeCtx := context.Background()

		if err := h.router.Route(routeCtx, eventType, deliveryID, body); err != nil {
			slog.Error("github webhook: routing failed",
				"event_type", eventType,
				"delivery_id", deliveryID,
				"error", err,
			)
			// Routing failure is logged but cannot affect the HTTP response
			// (already sent 202). GitHub will NOT retry since we returned 202.
			// The reconciler (Section 19) catches these misses.
		}
	}()
}

// validateSignature verifies the HMAC-SHA256 signature GitHub attaches to
// every webhook delivery. The signature format is "sha256=<hex>".
//
// Security note: use hmac.Equal for constant-time comparison to prevent
// timing attacks. Never use == or bytes.Equal on HMAC values.
func (h *WebhookHandler) validateSignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Signature format: "sha256=<64-char hex>"
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}

	expectedHex := strings.TrimPrefix(signature, prefix)
	expected, err := hex.DecodeString(expectedHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	computed := mac.Sum(nil)

	return hmac.Equal(computed, expected)
}

// ValidateSignatureExported is an exported version of validateSignature for testing
func (h *WebhookHandler) ValidateSignatureExported(body []byte, signature string) bool {
	return h.validateSignature(body, signature)
}

// ComputeTestSignature computes a test signature for testing
func ComputeTestSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// NewTestWebhookHandler creates a WebhookHandler for testing
func NewTestWebhookHandler(secret string) *WebhookHandler {
	return &WebhookHandler{
		secret: secret,
	}
}

// Alias for tests
var newTestWebhookHandler = NewTestWebhookHandler
var computeTestSignature = ComputeTestSignature

// NewEventRouter creates an EventRouter
func NewEventRouter(
	installRepo InstallationRepository,
	deployProcessor interface{},
	prProcessor interface{},
	oauthHandler *OAuthHandler,
	queue interface{},
) *EventRouter {
	return &EventRouter{
		installRepo:     installRepo,
		deployProcessor: deployProcessor,
		prProcessor:     prProcessor,
		oauthHandler:    oauthHandler,
		queue:           queue,
	}
}

// EventRouter dispatches validated webhook payloads to the correct processor.
type EventRouter struct {
	installRepo     InstallationRepository
	deployProcessor interface{} // DeployProcessor
	prProcessor     interface{} // PRProcessor
	oauthHandler    *OAuthHandler
	queue           interface{} // JobQueue
}

func (r *EventRouter) Route(ctx context.Context, eventType, deliveryID string, body []byte) error {
	switch eventType {

	case "push":
		// Push events are the primary deploy signal.
		// We enqueue for async processing — parsing and DB writes happen in the worker.
		return nil // TODO: implement using r.queue.EnqueueDeployIngestion

	case "deployment":
		// Structured deployment events from the GitHub Deployments API.
		return nil // TODO: implement

	case "deployment_status":
		// Outcome of a deployment — updates the DeployEvent status.
		return nil // TODO: implement

	case "workflow_run":
		// Gate 2: Ingests GitHub Actions workflow completions as deploy signals.
		return nil // TODO: implement

	case "check_suite":
		// Gate 2: Ingests CI check status for commits.
		return nil // TODO: implement

	case "pull_request":
		// PR events — stored for risk scoring (was there a PR for this commit?)
		// and ticket linkage detection.
		return nil // TODO: implement using r.prProcessor.StorePRMetadata

	case "installation":
		// App installed or uninstalled.
		var payload struct {
			Action       string                 `json:"action"`
			Installation *gogithub.Installation `json:"installation"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return fmt.Errorf("parse installation event: %w", err)
		}
		if payload.Action == "deleted" || payload.Action == "suspend" {
			return r.oauthHandler.HandleUninstall(ctx, payload.Installation.GetID())
		}
		return nil

	case "installation_repositories":
		// Repositories added to or removed from the installation.
		// In Gate 1 we log this but take no action — all repos in the installation
		// are already covered by the installation-level webhook.
		slog.Info("installation_repositories event",
			"delivery_id", deliveryID,
			"note", "no action required — installation-level coverage",
		)
		return nil

	case "member":
		// Org member added or removed — used for self-approval detection enrichment.
		slog.Info("member event received", "delivery_id", deliveryID)
		return nil

	case "ping":
		// GitHub sends a ping when the webhook is first configured.
		// No action needed — the 202 response is sufficient.
		slog.Info("github webhook ping received", "delivery_id", deliveryID)
		return nil

	default:
		// Unknown event type — log and ignore.
		// This happens when GitHub adds new event types we have not subscribed to yet.
		slog.Warn("github webhook: unhandled event type",
			"event_type", eventType,
			"delivery_id", deliveryID,
		)
		return nil
	}
}
