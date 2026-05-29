// internal/integrations/github/reconciler.go

package github

import (
	"context"
	"log/slog"
	"time"
)

// Reconciler periodically fetches deployment records from the GitHub
// Deployments API and backfills any that are missing from DeployGuard.
//
// The GitHub Deployments API only covers structured deployments (created via
// the API or by GitHub Actions with the environment setting). Push-based
// deployments without the Deployments API are not reconcilable — they rely
// on webhooks only. This is acceptable: push-based deployments are a fallback;
// Deployments API usage is encouraged for production systems.
type Reconciler struct {
	enricher      *EnrichmentClient
	installRepo   InstallationRepository
	deployService interface{} // deploy.Service
}

func NewReconciler(
	enricher *EnrichmentClient,
	installRepo InstallationRepository,
	deployService interface{},
) *Reconciler {
	return &Reconciler{
		enricher:      enricher,
		installRepo:   installRepo,
		deployService: deployService,
	}
}

// Start runs the reconciler on a nightly schedule.
// Called from main.go in the errgroup alongside other workers.
func (r *Reconciler) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(24 * time.Hour): // Run daily
			if err := r.reconcileAll(ctx); err != nil {
				slog.Error("reconciliation failed", "error", err)
			}
		}
	}
}

func (r *Reconciler) reconcileAll(ctx context.Context) error {
	// TODO: implement
	return nil
}

// func (r *Reconciler) reconcileRepo(
// 	ctx context.Context,
// 	install *GitHubInstallation,
// 	repoName, repoFullName string,
// ) error {
// 	// Fetch deployments from the last 7 days from GitHub
// 	since := time.Now().UTC().AddDate(0, 0, -7)
//
// 	deployments, err := r.enricher.GetDeploymentsByRepo(
// 		ctx,
// 		install.InstallationID,
// 		install.AccountLogin,
// 		repoName,
// 		&gogithub.DeploymentsListOptions{
// 			Environment: "production",
// 			ListOptions: gogithub.ListOptions{PerPage: 100},
// 		},
// 	)
// 	if err != nil {
// 		return fmt.Errorf("list github deployments for %s: %w", repoFullName, err)
// 	}
//
// 	backfilled := 0
// 	for _, d := range deployments {
// 		if d.GetCreatedAt().Time.Before(since) {
// 			continue
// 		}
//
// 		// Check if we already have this deployment
// 		exists, err := r.deployService.ExistsByGitHubDeploymentID(ctx,
// 			install.OrgID, d.GetID())
// 		if err != nil {
// 			return fmt.Errorf("check deployment %d: %w", d.GetID(), err)
// 		}
// 		if exists {
// 			continue // already ingested via webhook
// 		}
//
// 		// Backfill the missing deployment
// 		deploymentID := d.GetID()
// 		_, err = r.deployService.IngestFromWebhook(ctx, deploy.IngestRequest{
// 			OrgID:              install.OrgID,
// 			Source:             "github_reconciler",
// 			RepoOwner:          install.AccountLogin,
// 			RepoName:           repoName,
// 			CommitSHA:          d.GetSHA(),
// 			Branch:             d.GetRef(),
// 			Environment:        d.GetEnvironment(),
// 			AuthorLogin:        d.GetCreator().GetLogin(),
// 			TriggeredAt:        d.GetCreatedAt().Time,
// 			GitHubDeploymentID: &deploymentID,
// 		})
// 		if err != nil {
// 			slog.Error("reconciler: failed to backfill deployment",
// 				"github_deployment_id", d.GetID(),
// 				"repo", repoFullName,
// 				"error", err,
// 			)
// 			continue
// 		}
//
// 		backfilled++
// 		slog.Info("reconciler: backfilled missing deployment",
// 			"github_deployment_id", d.GetID(),
// 			"repo", repoFullName,
// 			"environment", d.GetEnvironment(),
// 		)
// 	}
//
// 	if backfilled > 0 {
// 		slog.Info("reconciler: repo reconciliation complete",
// 			"repo", repoFullName,
// 			"backfilled", backfilled,
// 		)
// 	}
//
// 	return nil
// }
