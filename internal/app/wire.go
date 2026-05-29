package app

import (
	"context"
	"fmt"

	"github.com/samaasi/watchnoc/internal/config"

	githubintegration "github.com/samaasi/watchnoc/internal/integrations/github"
	jiraintegration "github.com/samaasi/watchnoc/internal/integrations/jira"
	trellointegration "github.com/samaasi/watchnoc/internal/integrations/trello"
)

// Wire assembles the application's dependencies.
func Wire(
	ctx context.Context,
	cfg *config.Config,
	db interface{},
	orgService interface{},
	responder interface{},
	deployService interface{},
	prProcessor interface{},
	jobQueue interface{},
	ticketService interface{},
	ticketRepo interface{},
) (*App, func(), error) {

	cleanup := func() {}

	// --- GitHub App Client ---
	privateKey, err := cfg.GitHub.DecodedPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("decode github private key: %w", err)
	}

	githubAppClient, err := githubintegration.NewAppClient(
		cfg.GitHub.AppID,
		privateKey,
		cfg.GitHub.BaseURL,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("init github app client: %w", err)
	}

	// --- GitHub Enrichment Client ---
	githubEnrichment := githubintegration.NewEnrichmentClient(githubAppClient)

	// --- GitHub Installation Repository ---
	// Type casting db for now, as the actual type is defined in the storage package
	githubInstallRepo := githubintegration.NewInstallationRepository(db)

	// --- GitHub OAuth Handler ---
	githubOAuth := githubintegration.NewOAuthHandler(
		githubAppClient,
		githubInstallRepo,
		orgService,
		responder,
		cfg.GitHub.ClientID,
		cfg.GitHub.ClientSecret,
	)

	// --- GitHub Webhook Handler ---
	eventRouter := githubintegration.NewEventRouter(
		githubInstallRepo,
		deployService,
		prProcessor,
		githubOAuth,
		jobQueue,
	)

	githubWebhook := githubintegration.NewWebhookHandler(
		cfg.GitHub.WebhookSecret,
		eventRouter,
		responder,
		githubInstallRepo,
	)

	// --- GitHub Reconciler Worker ---
	githubReconciler := githubintegration.NewReconciler(
		githubEnrichment,
		githubInstallRepo,
		deployService,
	)

	// --- Trello Client ---
	trelloClient := trellointegration.NewClient(
		cfg.Trello.APIKey,
		cfg.Trello.BaseURL,
	)

	// --- Trello Installation Repository ---
	trelloInstallRepo := trellointegration.NewInstallationRepository(db)

	// --- Trello Enrichment Client ---
	trelloEnrichment := trellointegration.NewEnrichmentClient(
		trelloClient,
		trelloInstallRepo,
		cfg.Trello.WebhookCallbackURL,
	)

	// --- Trello OAuth Handler ---
	trelloOAuth := trellointegration.NewOAuthHandler(
		trelloClient,
		trelloInstallRepo,
		orgService,
		responder,
		cfg.Trello.APIKey,
		cfg.Trello.APISecret,
		cfg.Trello.CallbackURL,
	)

	// --- Trello Event Router ---
	trelloEventRouter := trellointegration.NewEventRouter(
		trelloInstallRepo,
		ticketService,
		jobQueue,
	)

	// --- Trello Webhook Handler ---
	trelloWebhook := trellointegration.NewWebhookHandler(
		trelloEventRouter,
		trelloInstallRepo,
		responder,
	)

	// --- Trello Reconciler ---
	trelloReconciler := trellointegration.NewReconciler(
		trelloEnrichment,
		trelloInstallRepo,
		ticketRepo,
	)

	// --- Jira Token Store ---
	jiraTokenStore := jiraintegration.NewTokenStore(db)

	// --- Jira Installation Repository ---
	jiraInstallRepo := jiraintegration.NewInstallationRepository(db)

	// --- Jira Client Factory ---
	jiraClientFactory := jiraintegration.NewClientFactory(
		cfg.Jira.ClientID,
		cfg.Jira.ClientSecret,
		jiraTokenStore,
		jiraInstallRepo,
	)

	// --- Jira OAuth Handler ---
	jiraOAuth := jiraintegration.NewOAuthHandler(
		cfg.Jira.ClientID,
		cfg.Jira.ClientSecret,
		cfg.Jira.CallbackURL,
		jiraTokenStore,
		jiraInstallRepo,
		orgService,
		responder,
	)

	// --- Jira Issue Resolver ---
	jiraResolver := jiraintegration.NewIssueResolver(jiraClientFactory)

	// --- Jira Linkage Repository ---
	jiraLinkRepo := jiraintegration.NewLinkageRepository(db)

	// --- Jira Linkage Engine ---
	jiraLinkage := jiraintegration.NewLinkageEngine(jiraResolver, jiraLinkRepo, deployService)

	// --- Jira Event Router ---
	jiraEventRouter := jiraintegration.NewEventRouter(ticketService, jobQueue)

	// --- Jira Webhook Handler ---
	jiraWebhook := jiraintegration.NewWebhookHandler(
		cfg.Jira.WebhookSecret,
		jiraEventRouter,
		responder,
		jiraInstallRepo,
	)

	// --- Jira Reconciler ---
	jiraReconciler := jiraintegration.NewReconciler(jiraLinkage, deployService, jiraInstallRepo)

	return &App{
		GitHubWebhook:     githubWebhook,
		GitHubOAuth:       githubOAuth,
		GitHubReconciler:  githubReconciler,
		TrelloWebhook:     trelloWebhook,
		TrelloOAuth:       trelloOAuth,
		TrelloEnrichment:  trelloEnrichment,
		TrelloInstallRepo: trelloInstallRepo,
		TrelloReconciler:  trelloReconciler,
		JiraWebhook:       jiraWebhook,
		JiraOAuth:         jiraOAuth,
		JiraReconciler:    jiraReconciler,
		JiraLinkage:       jiraLinkage,
		JiraInstallRepo:   jiraInstallRepo,
	}, cleanup, nil
}
