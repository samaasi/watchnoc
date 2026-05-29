
package linear

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

// OAuthHandler manages the Linear OAuth 2.0 installation flow.
type OAuthHandler struct {
	clientID     string
	clientSecret string
	callbackURL  string
	tokenStore   TokenStore
	installRepo  InstallationRepository
	orgService   interface{}
	responder    interface{}
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
func (h *OAuthHandler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	state := generateOAuthState()
	setOAuthStateCookie(w, state)

	params := url.Values{}
	params.Set("client_id", h.clientID)
	params.Set("redirect_uri", h.callbackURL)
	params.Set("response_type", "code")
	params.Set("scope", "read webhooks")
	params.Set("state", state)

	authorizeURL := "https://linear.app/oauth/authorize?" + params.Encode()
	http.Redirect(w, r, authorizeURL, http.StatusFound)
}

// HandleCallback processes the OAuth callback from Linear.
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	state := r.URL.Query().Get("state")
	if !validateOAuthState(r, state) {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		errorParam := r.URL.Query().Get("error")
		slog.Warn("linear oauth: user declined authorization", "error", errorParam)
		http.Redirect(w, r, "/settings?linear=declined", http.StatusFound)
		return
	}

	tokens, err := h.exchangeCode(ctx, code)
	if err != nil {
		slog.Error("linear oauth: code exchange failed", "error", err)
		http.Error(w, "code exchange failed", http.StatusInternalServerError)
		return
	}

	workspace, err := h.fetchWorkspace(ctx, tokens.AccessToken)
	if err != nil {
		slog.Error("linear oauth: failed to fetch workspace", "error", err)
		http.Error(w, "failed to fetch workspace", http.StatusInternalServerError)
		return
	}

	orgID := org.IDFromContext(ctx)

	if err := h.tokenStore.SaveTokens(ctx, orgID, tokens); err != nil {
		http.Error(w, "failed to save tokens", http.StatusInternalServerError)
		return
	}

	install := &Installation{
		OrgID:           orgID,
		LinearWorkspace: workspace,
		AccessToken:     tokens.AccessToken,
	}
	if err := h.installRepo.Create(ctx, install); err != nil {
		http.Error(w, "failed to create installation", http.StatusInternalServerError)
		return
	}

	slog.Info("linear connected", "org_id", orgID, "workspace", workspace)
	http.Redirect(w, r, "/settings?linear=connected", http.StatusFound)
}

// exchangeCode exchanges an OAuth authorization code for tokens.
func (h *OAuthHandler) exchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     h.clientID,
		"client_secret": h.clientSecret,
		"code":          code,
		"redirect_uri":  h.callbackURL,
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.linear.app/oauth/token", bytes.NewReader(data))
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

// fetchWorkspace retrieves the Linear workspace name for the authenticated user.
func (h *OAuthHandler) fetchWorkspace(ctx context.Context, accessToken string) (string, error) {
	query := `
		query GetWorkspace {
			viewer {
				organization {
					name
				}
			}
		}
	`

	payload := map[string]any{"query": query}
	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.linear.app/graphql", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Viewer struct {
				Organization struct {
					Name string `json:"name"`
				} `json:"organization"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Data.Viewer.Organization.Name, nil
}

// Stub functions for OAuth state management.
func generateOAuthState() string {
	return "stub-state"
}

func setOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:  "linear_oauth_state",
		Value: state,
	})
}

func validateOAuthState(r *http.Request, state string) bool {
	cookie, err := r.Cookie("linear_oauth_state")
	if err != nil {
		return false
	}
	return cookie.Value == state
}
