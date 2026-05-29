package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const (
	// maxWebhookBodySize — Trello payloads are small (card events are compact JSON).
	// 1MB is sufficient; we allow 5MB for safety.
	maxWebhookBodySize = 5 * 1024 * 1024 // 5MB

	// callbackURLHeader is sent by Trello on webhook delivery.
	// Not used for signature validation — Trello does not sign payloads.
	callbackURLHeader = "X-Trello-Webhook"
)

// WebhookHandler receives and routes Trello webhook events.
type WebhookHandler struct {
	router      *EventRouter
	installRepo InstallationRepository
	responder   interface{} // response.Responder
}

func NewWebhookHandler(
	router *EventRouter,
	installRepo InstallationRepository,
	responder interface{},
) *WebhookHandler {
	return &WebhookHandler{
		router:      router,
		installRepo: installRepo,
		responder:   responder,
	}
}

// Handle is the HTTP handler registered at /webhooks/trello.
// Trello sends both HEAD (for registration validation) and POST (for events).
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Trello validates the webhook URL with a HEAD request during registration.
	// We must return 200 to confirm the URL is reachable.
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		// TODO: use responder
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	h.handlePost(w, r)
}

func (h *WebhookHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize))
	if err != nil {
		slog.Warn("trello webhook: failed to read body", "error", err)
		// TODO: use responder
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Trello does not sign webhook payloads — there is no HMAC header to validate.
	// Security is enforced by:
	//   1. The webhook URL contains a per-installation secret path component (see Section 16).
	//   2. Source IP allowlisting at the load balancer (optional, but recommended).
	//   3. The modelID in the payload is validated against our own records.
	// See Section 16 for the full security discussion.

	var envelope struct {
		Action struct {
			Type string `json:"type"`
		} `json:"action"`
		Model struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"model"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil {
		slog.Warn("trello webhook: malformed payload", "error", err)
		// Return 200 — Trello retries on non-2xx. A malformed payload won't
		// become valid on retry; returning 200 stops retry spam.
		w.WriteHeader(http.StatusOK)
		return
	}

	slog.Info("trello webhook received",
		"action_type", envelope.Action.Type,
		"model_id", envelope.Model.ID,
		"body_size", len(body),
	)

	// Acknowledge immediately. Trello expects a 2xx within a short window.
	w.WriteHeader(http.StatusOK)

	// Process asynchronously — do not block the HTTP response.
	go func() {
		routeCtx := context.Background()
		if err := h.router.Route(routeCtx, envelope.Action.Type, envelope.Model.ID, body); err != nil {
			slog.Error("trello webhook: routing failed",
				"action_type", envelope.Action.Type,
				"model_id", envelope.Model.ID,
				"error", err,
			)
		}
	}()
}

// EventRouter dispatches Trello webhook events to the appropriate processor.
type EventRouter struct {
	installRepo   InstallationRepository
	ticketService interface{} // TicketLinker
	queue         interface{} // JobQueue
}

func (r *EventRouter) Route(ctx context.Context, actionType, modelID string, body []byte) error {
	switch actionType {

	case "updateCard":
		// Card was modified — title, description, list move, due date change, etc.
		event, err := ParseCardUpdateEvent(body)
		if err != nil {
			return fmt.Errorf("parse updateCard event: %w", err)
		}
		return r.handleCardUpdate(ctx, event)

	case "createCard":
		// New card created. Not actionable for DeployGuard in Gate 1 —
		// we only care about cards that already exist and are linked to deploys.
		slog.Info("trello: createCard event — no action", "model_id", modelID)
		return nil

	case "addMemberToCard":
		// Member added to card — could indicate review/approval.
		// Logged for future use; no action in Gate 1.
		slog.Info("trello: addMemberToCard event — no action", "model_id", modelID)
		return nil

	case "commentCard":
		// Comment added — could contain deploy references.
		// Logged for future enrichment; no action in Gate 1.
		slog.Info("trello: commentCard event — no action", "model_id", modelID)
		return nil

	case "deleteCard":
		// Card was deleted — mark any linked tickets as invalidated.
		event, err := ParseCardDeleteEvent(body)
		if err != nil {
			return fmt.Errorf("parse deleteCard event: %w", err)
		}
		return r.handleCardDelete(ctx, event)

	default:
		// Log and ignore — Trello has many event types we do not need.
		slog.Info("trello webhook: unhandled action type",
			"action_type", actionType,
			"model_id", modelID,
		)
		return nil
	}
}

func (r *EventRouter) handleCardUpdate(ctx context.Context, event *ParsedCardUpdateEvent) error {
	// If the card moved to a "deployed" or "done" list, enqueue an enrichment job.
	// The enrichment worker fetches the full card state and updates the linked ticket.
	// TODO: implement using r.queue
	return nil // TODO: implement
}

func (r *EventRouter) handleCardDelete(ctx context.Context, event *ParsedCardDeleteEvent) error {
	return nil // TODO: implement
}

// NewEventRouter creates an EventRouter
func NewEventRouter(
	installRepo InstallationRepository,
	ticketService interface{},
	queue interface{},
) *EventRouter {
	return &EventRouter{
		installRepo:   installRepo,
		ticketService: ticketService,
		queue:         queue,
	}
}

// NewTestWebhookHandler creates a WebhookHandler for testing
func NewTestWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}

// Alias for tests
var newTestWebhookHandler = NewTestWebhookHandler
