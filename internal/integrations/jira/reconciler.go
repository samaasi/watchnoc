package jira

import (
	"context"
	"log/slog"
)

// Reconciler periodically checks for unlinked deploys and attempts to link them to Jira issues.
type Reconciler struct {
	linkageEngine *LinkageEngine
	deployService interface{} // deploy.Service (stub)
	installRepo   InstallationRepository
}

// NewReconciler creates a new Reconciler.
func NewReconciler(linkageEngine *LinkageEngine, deployService interface{}, installRepo InstallationRepository) *Reconciler {
	return &Reconciler{
		linkageEngine: linkageEngine,
		deployService: deployService,
		installRepo:   installRepo,
	}
}

// Start runs the reconciler.
func (r *Reconciler) Start(ctx context.Context) error {
	// TODO: implement actual reconciliation logic
	slog.Info("jira: reconciler started")
	return nil
}
