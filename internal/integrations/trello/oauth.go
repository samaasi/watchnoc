// internal/integrations/trello/oauth.go

package trello

import (
	"context"
	"net/http"
)

// OAuthHandler manages the Trello OAuth 1.0a installation flow.
type OAuthHandler struct {
	client      *Client
	installRepo InstallationRepository
	orgService  interface{} // org.Service
	responder   interface{} // response.Responder
	apiKey      string
	apiSecret   string
	callbackURL string
}

func NewOAuthHandler(
	client *Client,
	installRepo InstallationRepository,
	orgService interface{},
	responder interface{},
	apiKey, apiSecret, callbackURL string,
) *OAuthHandler {
	return &OAuthHandler{
		client:      client,
		installRepo: installRepo,
		orgService:  orgService,
		responder:   responder,
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		callbackURL: callbackURL,
	}
}

// HandleAuthorise redirects the user to Trello's OAuth authorisation page.
//
// Trello OAuth 1.0a flow:
//  1. GET  /trello/authorise        → this handler → redirect to Trello
//  2. user approves in Trello
//  3. GET  /trello/callback?token=  → HandleCallback
//
// Route: GET /trello/authorise
func (h *OAuthHandler) HandleAuthorise(w http.ResponseWriter, r *http.Request) {
	// TODO: implement actual functionality
	http.Redirect(w, r, "/settings?trello=connected", http.StatusFound)
}

// HandleCallback receives the token after the user approves DeployGuard in Trello.
// Trello's simplified OAuth passes the token directly as a query parameter.
//
// Route: GET /trello/callback
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// TODO: implement actual functionality
	http.Redirect(w, r, "/settings?trello=connected", http.StatusFound)
}

// HandleDisconnect revokes the Trello token and removes the installation record.
// This does NOT call Trello's revoke endpoint — Trello tokens are user-managed.
// We simply delete our copy and unregister all webhooks we registered.
//
// Route: DELETE /trello/connection
func (h *OAuthHandler) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	// TODO: implement actual functionality
	w.WriteHeader(http.StatusNoContent)
}

func (h *OAuthHandler) unregisterAllWebhooks(ctx context.Context, install *Installation) error {
	return nil // TODO: implement
}

// NewInstallationRepository creates a new InstallationRepository (stub)
func NewInstallationRepository(db interface{}) InstallationRepository {
	// TODO: implement real repository
	return &stubTrelloInstallationRepository{}
}

type stubTrelloInstallationRepository struct{}

func (s *stubTrelloInstallationRepository) Create(ctx context.Context, install *Installation) error {
	return nil
}
func (s *stubTrelloInstallationRepository) Update(ctx context.Context, install *Installation) error {
	return nil
}
func (s *stubTrelloInstallationRepository) FindByOrgID(ctx context.Context, orgID uint64) (*Installation, error) {
	return nil, nil
}
func (s *stubTrelloInstallationRepository) FindWebhookByBoardID(ctx context.Context, orgID uint64, boardID string) (*WebhookRecord, error) {
	return nil, nil
}
func (s *stubTrelloInstallationRepository) ListWebhooks(ctx context.Context, orgID uint64) ([]*WebhookRecord, error) {
	return nil, nil
}
func (s *stubTrelloInstallationRepository) SaveWebhook(ctx context.Context, record *WebhookRecord) error {
	return nil
}
func (s *stubTrelloInstallationRepository) DeleteWebhook(ctx context.Context, record *WebhookRecord) error {
	return nil
}
func (s *stubTrelloInstallationRepository) ListActive(ctx context.Context) ([]*Installation, error) {
	return nil, nil
}
func (s *stubTrelloInstallationRepository) MarkTokenExpired(ctx context.Context, orgID uint64) error {
	return nil
}
func (s *stubTrelloInstallationRepository) UpdateWebhookID(ctx context.Context, oldID, newID string) error {
	return nil
}
