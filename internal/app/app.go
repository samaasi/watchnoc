package app

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/samaasi/watchnoc/internal/config"
	"github.com/samaasi/watchnoc/internal/store"

	githubintegration "github.com/samaasi/watchnoc/internal/integrations/github"
	jiraintegration "github.com/samaasi/watchnoc/internal/integrations/jira"
	trellointegration "github.com/samaasi/watchnoc/internal/integrations/trello"
)

type App struct {
	DB  *gorm.DB
	RDB *redis.Client

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

func NewApp(db *gorm.DB, rdb *redis.Client) (*App, error) {
	return &App{
		DB:  db,
		RDB: rdb,
	}, nil
}

func (a *App) Close() error {
	var errs []error

	if a.DB != nil {
		sqlDB, err := a.DB.DB()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get DB instance: %w", err))
		} else {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close DB: %w", err))
			}
		}
	}

	if a.RDB != nil {
		if err := a.RDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// NewAppFromConfig is the entrypoint used by cmd/server/main.go to initialize the App.
func NewAppFromConfig(ctx context.Context, cfg *config.Config) (*App, func(), error) {
	// Run database migrations first
	if err := store.RunMigrations(cfg.Database, ""); err != nil {
		return nil, nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize PostgreSQL
	db, err := store.NewPostgres(cfg.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init postgres: %w", err)
	}

	// Initialize Redis
	rdb, err := store.NewRedis(cfg.Redis)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init redis: %w", err)
	}

	// Create app
	app, err := NewApp(db, rdb)
	if err != nil {
		// Cleanup connections if app creation fails
		app.Close()
		return nil, nil, err
	}

	cleanup := func() {
		app.Close()
	}

	return app, cleanup, nil
}
