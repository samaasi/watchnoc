package linear

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
)

const maxWebhookBodySize = 5 * 1024 * 1024

// WebhookHandler receives and routes Linear webhook events.
type WebhookHandler struct {
	secret      string
	router      *EventRouter
	responder   interface{}
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

// Handle is the HTTP handler for Linear webhooks.
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}

	signature := r.Header.Get("Linear-Signature")
	if !h.validateSignature(body, signature) {
		slog.Warn("linear webhook: invalid signature", "remote_addr", r.RemoteAddr)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)

	go func() {
		routeCtx := ctx
		if err := h.router.Route(routeCtx, body); err != nil {
			slog.Error("linear webhook: routing failed", "error", err)
		}
	}()
}

func (h *WebhookHandler) validateSignature(payload []byte, signatureHeader string) bool {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedSignature), []byte(signatureHeader))
}

// EventRouter dispatches Linear webhook events.
type EventRouter struct {
	linkageService interface{}
	queue          interface{}
}

// NewEventRouter creates a new EventRouter.
func NewEventRouter(linkageService interface{}, queue interface{}) *EventRouter {
	return &EventRouter{
		linkageService: linkageService,
		queue:          queue,
	}
}

func (r *EventRouter) Route(ctx context.Context, body []byte) error {
	// TODO: parse and route the Linear webhook event
	_ = body // Temporary use to avoid unused variable
	return nil
}
