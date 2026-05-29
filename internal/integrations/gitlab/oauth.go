package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// OAuthHandler handles the GitLab OAuth installation flow.
type OAuthHandler struct {
	clientFactory *ClientFactory
	installRepo   InstallationRepository
	responder     interface{} // response.Responder
	clientID      string
	clientSecret  string
	redirectURI   string
}

// NewOAuthHandler creates a new OAuthHandler.
func NewOAuthHandler(
	clientFactory *ClientFactory,
	installRepo InstallationRepository,
	orgService interface{},
	responder interface{},
	clientID, clientSecret, redirectURI string,
) *OAuthHandler {
	return &OAuthHandler{
		clientFactory: clientFactory,
		installRepo:   installRepo,
		responder:     responder,
		clientID:      clientID,
		clientSecret:  clientSecret,
		redirectURI:   redirectURI,
	}
}

// HandleConnect begins the OAuth flow.
func (h *OAuthHandler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	// TODO: Generate a CSRF state token and store it in the session
	state := "stub-state" // Placeholder
	setOAuthStateCookie(w, state)

	params := r.URL.Query()
	params.Set("client_id", h.clientID)
	params.Set("redirect_uri", h.redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "api read_user read_api read_repository write_repository")
	params.Set("state", state)

	http.Redirect(w, r, "https://gitlab.com/oauth/authorize?"+params.Encode(), http.StatusFound)
}

// HandleCallback processes the OAuth callback from GitLab.
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Validate CSRF state
	state := r.URL.Query().Get("state")
	if !validateOAuthState(r, state) {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Warn("gitlab oauth: user declined authorization", "error", r.URL.Query().Get("error"))
		http.Redirect(w, r, "/settings?gitlab=declined", http.StatusFound)
		return
	}

	tokens, err := h.exchangeCode(ctx, code)
	if err != nil {
		slog.Error("gitlab oauth: code exchange failed", "error", err)
		http.Error(w, "code exchange failed", http.StatusInternalServerError)
		return
	}

	// TODO: Fetch user and group info to create installation
	slog.Info("gitlab connected", "tokens_expire_at", tokens.ExpiresAt)
	http.Redirect(w, r, "/settings?gitlab=connected", http.StatusFound)
}

func (h *OAuthHandler) exchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	payload := map[string]string{
		"client_id":     h.clientID,
		"client_secret": h.clientSecret,
		"code":          code,
		"grant_type":    "authorization_code",
		"redirect_uri":  h.redirectURI,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://gitlab.com/oauth/token", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for k, v := range payload {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("code exchange failed: status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}

// Stub functions for OAuth state management
func generateOAuthState() string {
	return "stub-state"
}

func setOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:  "gitlab_oauth_state",
		Value: state,
	})
}

func validateOAuthState(r *http.Request, state string) bool {
	cookie, err := r.Cookie("gitlab_oauth_state")
	if err != nil {
		return false
	}
	return cookie.Value == state
}
