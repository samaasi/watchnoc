package trello

import (
	"context"
	"log/slog"
	"time"
)

// Reconciler maintains liveness of Trello webhooks and card metadata.
type Reconciler struct {
	enricher    *EnrichmentClient
	installRepo InstallationRepository
	ticketRepo  interface{} // LinkedTicketRepository
}

func NewReconciler(
	enricher *EnrichmentClient,
	installRepo InstallationRepository,
	ticketRepo interface{},
) *Reconciler {
	return &Reconciler{
		enricher:    enricher,
		installRepo: installRepo,
		ticketRepo:  ticketRepo,
	}
}

// Start runs the reconciler on a nightly schedule.
func (r *Reconciler) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(24 * time.Hour): // Run daily
			if err := r.reconcileAll(ctx); err != nil {
				slog.Error("trello reconciliation failed", "error", err)
			}
		}
	}
}

func (r *Reconciler) reconcileAll(ctx context.Context) error {
	// TODO: implement
	return nil
}
