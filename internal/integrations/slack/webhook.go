
package slack

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

const (
	maxWebhookBodySize = 5 * 1024 * 1024
)

// WebhookHandler receives and routes Slack webhook events (slash commands, interactive components).
type WebhookHandler struct {
	signingSecret string
	installRepo   InstallationRepository
	responder     interface{} // response.Responder
}

func NewWebhookHandler(
	signingSecret string,
	installRepo InstallationRepository,
	responder interface{},
) *WebhookHandler {
	return &WebhookHandler{
		signingSecret: signingSecret,
		installRepo:   installRepo,
		responder:     responder,
	}
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Query().Get("challenge") != "" {
		// URL verification for Slack's Events API
		var req struct {
			Challenge string `json:"challenge"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(req.Challenge))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize))
	if err != nil {
		slog.Warn("slack webhook: failed to read body", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !h.validateSignature(r, body) {
		slog.Warn("slack webhook: invalid signature")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)

	go func() {
		if err := h.processEvent(context.Background(), body); err != nil {
			slog.Error("slack webhook: processing failed", "error", err)
		}
	}()
}

func (h *WebhookHandler) validateSignature(r *http.Request, body []byte) bool {
	// TODO: implement Slack signature validation with hmac.SHA256
	return true
}

func (h *WebhookHandler) processEvent(ctx context.Context, body []byte) error {
	slog.Info("slack webhook received", "body_len", len(body))
	return nil
}
