// internal/integrations/github/oauth.go

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	gogithub "github.com/google/go-github/v62/github"
)

// OAuthHandler handles the GitHub App OAuth installation flow.
type OAuthHandler struct {
	appClient    *AppClient
	installRepo  InstallationRepository
	responder    interface{} // response.Responder
	clientID     string
	clientSecret string
}

// HandleInstall is the entry point when a user clicks "Install" on the
// GitHub App page. GitHub redirects here with ?installation_id=&setup_action=
//
// Route: GET /github/setup
func (h *OAuthHandler) HandleInstall(w http.ResponseWriter, r *http.Request) {
	// TODO: implement actual functionality using responder
	// For now, just redirect
	http.Redirect(w, r, "/settings?github=connected", http.StatusFound)
}

// HandleCallback handles the OAuth callback after a user authorises the App.
// Route: GET /github/callback
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// TODO: implement actual functionality
	http.Redirect(w, r, "/settings?github=connected", http.StatusFound)
}

// HandleUninstall is called by the installation.deleted webhook event.
// It marks the installation as revoked — data is NOT deleted.
// Called from webhook.go, not directly from an HTTP route.
func (h *OAuthHandler) HandleUninstall(ctx context.Context, installationID int64) error {
	if err := h.installRepo.MarkRevoked(ctx, installationID); err != nil {
		return fmt.Errorf("mark installation %d revoked: %w", installationID, err)
	}

	slog.Info("github app uninstalled", "installation_id", installationID)
	return nil
}

func (h *OAuthHandler) fetchInstallation(ctx context.Context, installationID int64) (*gogithub.Installation, error) {
	client, err := h.appClient.InstallationClient(ctx, installationID)
	if err != nil {
		return nil, err
	}

	installation, _, err := client.Apps.GetInstallation(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("get installation %d: %w", installationID, err)
	}

	return installation, nil
}

// NewOAuthHandler creates a new OAuthHandler
func NewOAuthHandler(
	appClient *AppClient,
	installRepo InstallationRepository,
	orgService interface{},
	responder interface{},
	clientID, clientSecret string,
) *OAuthHandler {
	return &OAuthHandler{
		appClient:    appClient,
		installRepo:  installRepo,
		responder:    responder,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// NewInstallationRepository creates a new InstallationRepository (stub)
func NewInstallationRepository(db interface{}) InstallationRepository {
	// TODO: implement real repository
	return &stubInstallationRepository{}
}

type stubInstallationRepository struct{}

func (s *stubInstallationRepository) Create(ctx context.Context, install *Installation) error {
	return nil
}
func (s *stubInstallationRepository) Update(ctx context.Context, install *Installation) error {
	return nil
}
func (s *stubInstallationRepository) FindByOrgID(ctx context.Context, orgID uint64) (*Installation, error) {
	return nil, nil
}
func (s *stubInstallationRepository) FindByRepoFullName(ctx context.Context, repoFullName string) (*Installation, error) {
	return nil, nil
}
func (s *stubInstallationRepository) ListActive(ctx context.Context) ([]*Installation, error) {
	return nil, nil
}
func (s *stubInstallationRepository) MarkRevoked(ctx context.Context, installationID int64) error {
	return nil
}

func exchangeCode(ctx context.Context, clientID, clientSecret, code string) (string, string, error) {
	// Standard OAuth2 code exchange
	// Returns: access_token, token_type, error
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token", nil)
	q := req.URL.Query()
	q.Set("client_id", clientID)
	q.Set("client_secret", clientSecret)
	q.Set("code", code)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	if result.Error != "" {
		return "", "", fmt.Errorf("github oauth: %s", result.Error)
	}

	return result.AccessToken, result.TokenType, nil
}
