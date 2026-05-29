package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/samaasi/watchnoc/internal/domain/org"
)

// OAuthHandler manages the Jira OAuth 2.0 (3LO) installation flow.
type OAuthHandler struct {
	clientID     string
	clientSecret string
	callbackURL  string
	tokenStore   TokenStore
	installRepo  InstallationRepository
	orgService   interface{} // org.Service (stub)
	responder    interface{} // response.Responder (stub)
}

// NewOAuthHandler creates a new OAuthHandler.
func NewOAuthHandler(clientID, clientSecret, callbackURL string, tokenStore TokenStore, installRepo InstallationRepository, orgService interface{}, responder interface{}) *OAuthHandler {
	return &OAuthHandler{
		clientID:     clientID,
		clientSecret: clientSecret,
		callbackURL:  callbackURL,
		tokenStore:   tokenStore,
		installRepo:  installRepo,
		orgService:   orgService,
		responder:    responder,
	}
}

// HandleConnect begins the OAuth flow.
// Redirects the user to Atlassian's authorisation page.
//
// Route: GET /jira/connect
func (h *OAuthHandler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	// Generate a CSRF state token and store it in the session.
	// The state is validated in the callback to prevent CSRF attacks.
	state := generateOAuthState()
	setOAuthStateCookie(w, state)

	params := url.Values{}
	params.Set("client_id", h.clientID)
	params.Set("redirect_uri", h.callbackURL)
	params.Set("response_type", "code")
	params.Set("scope", "read:jira-work read:jira-user read:me offline_access manage:jira-webhook")
	params.Set("state", state)
	params.Set("prompt", "consent") // Always show consent — ensures refresh token is issued

	authoriseURL := "https://auth.atlassian.com/authorize?" + params.Encode()
	http.Redirect(w, r, authoriseURL, http.StatusFound)
}

// HandleCallback processes the OAuth callback from Atlassian.
//
// Route: GET /jira/callback
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate CSRF state
	state := r.URL.Query().Get("state")
	if !validateOAuthState(r, state) {
		// TODO: use responder.Error
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		// User declined authorisation
		errorParam := r.URL.Query().Get("error")
		slog.Warn("jira oauth: user declined authorisation", "error", errorParam)
		http.Redirect(w, r, "/settings?jira=declined", http.StatusFound)
		return
	}

	// Exchange the code for tokens
	tokens, err := h.exchangeCode(ctx, code)
	if err != nil {
		slog.Error("jira oauth: code exchange failed", "error", err)
		// TODO: use responder.Error
		http.Error(w, "code exchange failed", http.StatusInternalServerError)
		return
	}

	// Fetch the list of Atlassian Cloud sites this token can access.
	// Each "site" corresponds to one Jira workspace (e.g., acme.atlassian.net).
	sites, err := h.fetchAccessibleResources(ctx, tokens.AccessToken)
	if err != nil {
		slog.Error("jira oauth: failed to fetch accessible resources", "error", err)
		// TODO: use responder.Error
		http.Error(w, "failed to fetch accessible resources", http.StatusInternalServerError)
		return
	}

	if len(sites) == 0 {
		// TODO: use responder.Error
		http.Error(w, "no Jira sites found", http.StatusBadRequest)
		return
	}

	// If multiple sites are accessible, we need the user to select one.
	// In Gate 1: use the first site. In Gate 2: redirect to a site picker.
	selectedSite := sites[0]

	tokens.CloudID = selectedSite.ID
	tokens.CloudName = selectedSite.Name

	orgID := org.IDFromContext(ctx)

	// Persist the tokens
	if err := h.tokenStore.SaveTokens(ctx, orgID, tokens); err != nil {
		// TODO: use responder.Error
		http.Error(w, "failed to save tokens", http.StatusInternalServerError)
		return
	}

	// Create the installation record
	record := &Installation{
		OrgID:     orgID,
		CloudID:   selectedSite.ID,
		CloudName: selectedSite.Name,
		CloudURL:  selectedSite.URL,
		Scope:     tokens.Scope,
	}
	if err := h.installRepo.Create(ctx, record); err != nil {
		// TODO: use responder.Error
		http.Error(w, "failed to create installation", http.StatusInternalServerError)
		return
	}

	slog.Info("jira connected",
		"org_id", orgID,
		"cloud_id", selectedSite.ID,
		"cloud_name", selectedSite.Name,
	)

	http.Redirect(w, r, "/settings?jira=connected", http.StatusFound)
}

// HandleDisconnect removes the Jira integration for the current org.
// Tokens are revoked with Atlassian and the installation record is soft-deleted.
//
// Route: DELETE /jira/disconnect
func (h *OAuthHandler) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := org.IDFromContext(ctx)

	tokens, err := h.tokenStore.GetTokens(ctx, orgID)
	if err != nil {
		// TODO: use responder.Error
		http.Error(w, "failed to get tokens", http.StatusInternalServerError)
		return
	}

	// Revoke the token with Atlassian — best effort
	if err := h.revokeToken(ctx, tokens.AccessToken); err != nil {
		slog.Warn("jira: failed to revoke token — continuing with disconnect",
			"org_id", orgID, "error", err,
		)
	}

	if err := h.installRepo.MarkRevoked(ctx, orgID); err != nil {
		// TODO: use responder.Error
		http.Error(w, "failed to mark installation revoked", http.StatusInternalServerError)
		return
	}

	slog.Info("jira disconnected", "org_id", orgID)
	// TODO: use responder.JSON
	w.WriteHeader(http.StatusOK)
}

// exchangeCode exchanges an OAuth authorisation code for access and refresh tokens.
func (h *OAuthHandler) exchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     h.clientID,
		"client_secret": h.clientSecret,
		"code":          code,
		"redirect_uri":  h.callbackURL,
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://auth.atlassian.com/oauth/token",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
		Scope:        result.Scope,
	}, nil
}

// fetchAccessibleResources returns the list of Atlassian Cloud sites the
// access token can reach. Used immediately after OAuth to determine cloudId.
func (h *OAuthHandler) fetchAccessibleResources(ctx context.Context, accessToken string) ([]AccessibleResource, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.atlassian.com/oauth/token/accessible-resources", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch accessible resources: %w", err)
	}
	defer resp.Body.Close()

	var resources []AccessibleResource
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, fmt.Errorf("decode accessible resources: %w", err)
	}

	return resources, nil
}

type AccessibleResource struct {
	ID     string   `json:"id"`   // cloudId
	Name   string   `json:"name"` // "acme.atlassian.net"
	URL    string   `json:"url"`  // "https://acme.atlassian.net"
	Scopes []string `json:"scopes"`
}

// Stub functions (TODO: implement session handling)
func generateOAuthState() string {
	// TODO: generate a random state string
	return "stub-state"
}

func setOAuthStateCookie(w http.ResponseWriter, state string) {
	// TODO: set state in secure cookie
	http.SetCookie(w, &http.Cookie{
		Name:  "jira_oauth_state",
		Value: state,
	})
}

func validateOAuthState(r *http.Request, state string) bool {
	// TODO: validate from cookie
	cookie, err := r.Cookie("jira_oauth_state")
	if err != nil {
		return false
	}
	return cookie.Value == state
}

func (h *OAuthHandler) revokeToken(ctx context.Context, accessToken string) error {
	// TODO: implement token revocation via Atlassian API
	return nil
}
