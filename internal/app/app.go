package app

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	githubintegration "github.com/samaasi/watchnoc/internal/integrations/github"
	jiraintegration "github.com/samaasi/watchnoc/internal/integrations/jira"
	trellointegration "github.com/samaasi/watchnoc/internal/integrations/trello"
)

type App struct {
	DB *gorm.DB

	// Integrations
	GitHubWebhook    *githubintegration.WebhookHandler
	GitHubOAuth      *githubintegration.OAuthHandler
	GitHubReconciler *githubintegration.Reconciler

	TrelloWebhook     *trellointegration.WebhookHandler
	TrelloOAuth       *trellointegration.OAuthHandler
	TrelloEnrichment  *trellointegration.EnrichmentClient
	TrelloInstallRepo trellointegration.InstallationRepository
	TrelloReconciler  *trellointegration.Reconciler

	JiraWebhook     *jiraintegration.WebhookHandler
	JiraOAuth       *jiraintegration.OAuthHandler
	JiraReconciler  *jiraintegration.Reconciler
	JiraLinkage     *jiraintegration.LinkageEngine
	JiraInstallRepo jiraintegration.InstallationRepository
}

func NewApp(db *gorm.DB) (*App, error) {
	return &App{
		DB: db,
	}, nil
}

func (a *App) Close() error {
	if a.DB == nil {
		return nil
	}
	sqlDB, err := a.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB instance: %w", err)
	}
	return sqlDB.Close()
}

// NewAppFromConfig is the entrypoint used by cmd/server/main.go to initialize the App.
func NewAppFromConfig(ctx context.Context, cfg interface{}) (*App, func(), error) {
	app, err := NewApp(nil)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		app.Close()
	}

	return app, cleanup, nil
}
