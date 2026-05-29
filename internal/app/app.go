package app

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/samaasi/watchnoc/internal/config"
	"github.com/samaasi/watchnoc/internal/platform/response"
	"github.com/samaasi/watchnoc/internal/store"

	"github.com/samaasi/watchnoc/internal/domain/approval"
	"github.com/samaasi/watchnoc/internal/domain/audit"
	"github.com/samaasi/watchnoc/internal/domain/auth"
	"github.com/samaasi/watchnoc/internal/domain/deploy"
	"github.com/samaasi/watchnoc/internal/domain/health"
	"github.com/samaasi/watchnoc/internal/domain/org"

	githubintegration "github.com/samaasi/watchnoc/internal/integrations/github"
	jiraintegration "github.com/samaasi/watchnoc/internal/integrations/jira"
	trellointegration "github.com/samaasi/watchnoc/internal/integrations/trello"
)

// deployReaderAdapter implements approval.DeployReader using deploy.Service
// This adapter keeps the domains decoupled by living in the app package,
// which imports both domains and handles the conversion.
type deployReaderAdapter struct {
	deploySvc deploy.Service
}

func (a *deployReaderAdapter) GetDeployForApproval(ctx context.Context, orgID, deployID uint64) (*approval.DeploySummary, error) {
	d, err := a.deploySvc.GetDeployForApproval(ctx, orgID, deployID)
	if err != nil {
		return nil, err
	}
	// Convert deploy.DeployForApproval to approval.DeploySummary
	return &approval.DeploySummary{
		ID:          d.ID,
		OrgID:       d.OrgID,
		AuthorLogin: d.AuthorLogin,
		RiskScore:   d.RiskScore,
		RiskLevel:   d.RiskLevel,
		Environment: d.Environment,
		RepoName:    d.RepoName,
		TriggeredAt: d.TriggeredAt,
	}, nil
}

type App struct {
	DB        *gorm.DB
	RDB       *redis.Client
	Responder *response.ChiResponder

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

	// Domain Handlers
	AuthHandler     *auth.Handler
	ApprovalHandler *approval.Handler
	AuditHandler    *audit.Handler
	DeployHandler   *deploy.Handler
	HealthHandler   *health.Handler
	OrgHandler      *org.Handler
}

func NewApp(
	db *gorm.DB,
	rdb *redis.Client,
	responder *response.ChiResponder,
	authHandler *auth.Handler,
	approvalHandler *approval.Handler,
	auditHandler *audit.Handler,
	deployHandler *deploy.Handler,
	healthHandler *health.Handler,
	orgHandler *org.Handler,
) (*App, error) {
	return &App{
		DB:              db,
		RDB:             rdb,
		Responder:       responder,
		AuthHandler:     authHandler,
		ApprovalHandler: approvalHandler,
		AuditHandler:    auditHandler,
		DeployHandler:   deployHandler,
		HealthHandler:   healthHandler,
		OrgHandler:      orgHandler,
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

func (a *App) RegisterRoutes(r chi.Router) {
	if a.AuthHandler != nil {
		a.AuthHandler.Register(r)
	}
	if a.ApprovalHandler != nil {
		a.ApprovalHandler.Register(r)
	}
	if a.AuditHandler != nil {
		a.AuditHandler.Register(r)
	}
	if a.DeployHandler != nil {
		a.DeployHandler.Register(r)
	}
	if a.HealthHandler != nil {
		a.HealthHandler.Register(r)
	}
	if a.OrgHandler != nil {
		a.OrgHandler.Register(r)
	}
}

// NewAppFromConfig is the entrypoint used by cmd/server/main.go to initialize the App.
func NewAppFromConfig(ctx context.Context, cfg *config.Config, responder *response.ChiResponder) (*App, func(), error) {
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

	// Initialize auth domain
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, cfg.Auth)
	authHandler := auth.NewHandler(authSvc, responder)

	// Initialize org domain
	orgRepo := org.NewRepository(db)
	orgSvc := org.NewService(orgRepo)
	orgHandler := org.NewHandler(orgSvc, responder)

	// Initialize health domain
	healthHandler := health.NewHandler(db, rdb)

	// Initialize deploy domain
	deployRepo := deploy.NewRepository(db)
	deploySvc := deploy.NewService(deployRepo)
	deployHandler := deploy.NewHandler(deploySvc, responder)

	// Initialize approval domain
	approvalRepo := approval.NewRepository(db)
	approvalSvc := approval.NewService(approvalRepo, &deployReaderAdapter{deploySvc: deploySvc})
	approvalHandler := approval.NewHandler(approvalSvc, responder)

	// Initialize audit domain
	auditRepo := audit.NewRepository(db)
	auditSvc := audit.NewService(auditRepo)
	auditHandler := audit.NewHandler(auditSvc, responder)

	// Create app
	app, err := NewApp(
		db,
		rdb,
		responder,
		authHandler,
		approvalHandler,
		auditHandler,
		deployHandler,
		healthHandler,
		orgHandler,
	)
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
