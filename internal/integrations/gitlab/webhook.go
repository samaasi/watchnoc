package gitlab

import (
	"context"
	"crypto/hmac"
	"io"
	"log/slog"
	"net/http"
)

const (
	// maxWebhookBodySize prevents memory exhaustion from malformed payloads.
	maxWebhookBodySize = 10 * 1024 * 1024 // 10MB

	// signatureHeader is the header GitLab sets on every webhook request.
	signatureHeader = "X-Gitlab-Token"
)

// WebhookHandler receives, validates, and routes GitLab webhook events.
type WebhookHandler struct {
	secret      string
	router      *EventRouter
	responder   interface{} // response.Responder
	installRepo InstallationRepository
}

// NewWebhookHandler creates a new WebhookHandler.
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

// Handle is the HTTP handler for GitLab webhooks.
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Read body with size limit
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize))
	if err != nil {
		slog.Warn("gitlab webhook: failed to read body", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Validate the secret token
	secret := r.Header.Get(signatureHeader)
	if !h.validateSecret(secret) {
		slog.Warn("gitlab webhook: invalid secret", "remote_addr", r.RemoteAddr)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Acknowledge immediately
	w.WriteHeader(http.StatusAccepted)

	// Process asynchronously
	go func() {
		routeCtx := context.Background()
		if err := h.router.Route(routeCtx, string(body)); err != nil {
			slog.Error("gitlab webhook: routing failed", "error", err)
		}
	}()
}

func (h *WebhookHandler) validateSecret(received string) bool {
	// GitLab uses a token-based verification by default
	if h.secret == "" {
		return true // No secret configured
	}
	return hmac.Equal([]byte(received), []byte(h.secret))
}

// NewEventRouter creates an EventRouter.
func NewEventRouter(
	installRepo InstallationRepository,
	deployProcessor interface{},
	mrProcessor interface{},
	oauthHandler *OAuthHandler,
	queue interface{},
) *EventRouter {
	return &EventRouter{
		installRepo:     installRepo,
		deployProcessor: deployProcessor,
		mrProcessor:     mrProcessor,
		oauthHandler:    oauthHandler,
		queue:           queue,
	}
}

// EventRouter dispatches validated webhook payloads to the correct processor.
type EventRouter struct {
	installRepo     InstallationRepository
	deployProcessor interface{}
	mrProcessor     interface{}
	oauthHandler    *OAuthHandler
	queue           interface{}
}

func (r *EventRouter) Route(ctx context.Context, payload string) error {
	slog.Info("gitlab webhook received", "payload_length", len(payload))
	return nil
}
