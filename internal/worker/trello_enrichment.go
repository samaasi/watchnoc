// internal/worker/trello_enrichment.go

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/samaasi/watchnoc/internal/domain/ticket"
	trellointegration "github.com/samaasi/watchnoc/internal/integrations/trello"
)

const (
	TaskTrelloEnrichment = "trello:card:enrich"
)

// TrelloEnrichmentPayload is the Asynq task payload.
type TrelloEnrichmentPayload struct {
	OrgID         uint64 `json:"org_id"`
	DeployEventID uint64 `json:"deploy_event_id"`
	CardShortID   string `json:"card_short_id"`
	// Source indicates how this enrichment was triggered.
	// "commit_message" | "webhook" | "manual"
	Source     string    `json:"source"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// TrelloEnrichmentWorker fetches Trello card details and creates or updates
// a LinkedTicket record for the given deploy event.
type TrelloEnrichmentWorker struct {
	enricher      *trellointegration.EnrichmentClient
	installRepo   trellointegration.InstallationRepository
	ticketService ticket.Service
	deliveryRepo  trellointegration.DeliveryRepository
}

// ProcessTask is the Asynq task handler.
func (w *TrelloEnrichmentWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload TrelloEnrichmentPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		slog.Error("trello enrichment: malformed payload", "error", err)
		return nil // do not retry malformed tasks
	}

	return w.processEnrichment(ctx, payload)
}

func (w *TrelloEnrichmentWorker) processEnrichment(
	ctx context.Context,
	payload TrelloEnrichmentPayload,
) error {
	install, err := w.installRepo.FindByOrgID(ctx, payload.OrgID)
	if err != nil {
		return fmt.Errorf("find trello installation for org %d: %w", payload.OrgID, err)
	}

	if install.RevokedAt != nil {
		slog.Warn("trello enrichment: installation revoked — skipping",
			"org_id", payload.OrgID,
			"card_short_id", payload.CardShortID,
		)
		return nil
	}

	// Fetch card details and ensure a board webhook exists.
	detail, err := w.enricher.GetCardAndEnsureWebhook(ctx, install, payload.CardShortID)
	if err != nil {
		// Non-fatal — the card may have been deleted or access revoked.
		// Log and do not retry indefinitely.
		slog.Warn("trello enrichment: failed to fetch card",
			"card_short_id", payload.CardShortID,
			"org_id", payload.OrgID,
			"error", err,
		)
		return fmt.Errorf("get trello card %s: %w", payload.CardShortID, err)
	}

	// Create or update the LinkedTicket record.
	if err := w.ticketService.LinkTicket(ctx, ticket.LinkRequest{
		DeployEventID: payload.DeployEventID,
		OrgID:         payload.OrgID,
		TicketURL:     detail.ShortURL,
		TicketKey:     detail.ShortID,
		TicketSource:  "trello",
		TicketMetadata: ticket.Metadata{
			Title:       detail.Name,
			Description: detail.Description,
			BoardName:   detail.BoardName,
			ListName:    detail.ListName,
			Labels:      labelsToStrings(detail.Labels),
		},
	}); err != nil {
		return fmt.Errorf("link trello ticket for deploy %d: %w",
			payload.DeployEventID, err)
	}

	slog.Info("trello enrichment: card linked to deploy",
		"deploy_event_id", payload.DeployEventID,
		"card_short_id", payload.CardShortID,
		"card_name", detail.Name,
		"source", payload.Source,
	)

	return nil
}

// NewTrelloEnrichmentTask creates an Asynq task with retry configuration.
func NewTrelloEnrichmentTask(payload TrelloEnrichmentPayload) (*asynq.Task, []asynq.Option) {
	data, _ := json.Marshal(payload)

	return asynq.NewTask(TaskTrelloEnrichment, data), []asynq.Option{
			// 3 retries — if the card is inaccessible after 3 attempts, give up.
			// Trello tokens rarely expire; repeated failures indicate a revoked token
			// or deleted card. Both should not be retried aggressively.
			asynq.MaxRetry(3),
			asynq.Deadline(time.Now().Add(4 * time.Hour)),
			// Unique within 60 seconds — prevents duplicate enrichment tasks
			// if the deploy ingestion worker enqueues the same card twice rapidly.
			asynq.Unique(60 * time.Second),
			asynq.Queue("default"),
		}
}

func labelsToStrings(labels []trellointegration.CardLabel) []string {
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		if l.Name != "" {
			out = append(out, l.Name)
		} else if l.Color != "" {
			out = append(out, l.Color)
		}
	}
	return out
}
