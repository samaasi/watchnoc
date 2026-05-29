package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// OAuthHandler manages the Slack OAuth 2.0 installation flow.
type OAuthHandler struct {
	clientFactory *ClientFactory
	installRepo   InstallationRepository
	orgService    interface{} // org.Service
	responder     interface{} // response.Responder
	clientID      string
	clientSecret  string
	redirectURI   string
}

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
		orgService:    orgService,
		responder:     responder,
		clientID:      clientID,
		clientSecret:  clientSecret,
		redirectURI:   redirectURI,
	}
}

// HandleBegin begins the OAuth 2.0 flow.
func (h *OAuthHandler) HandleBegin(w http.ResponseWriter, r *http.Request) {
	state := "slack_oauth_" + generateRandomState()
	setOAuthStateCookie(w, state)

	params := url.Values{}
	params.Set("client_id", h.clientID)
	params.Set("scope", "chat:write,channels:read,channels:join")
	params.Set("redirect_uri", h.redirectURI)
	params.Set("state", state)

	http.Redirect(w, r, "https://slack.com/oauth/v2/authorize?"+params.Encode(), http.StatusFound)
}

// HandleCallback receives the OAuth 2.0 code and exchanges it for a token.
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	state := r.URL.Query().Get("state")
	if !validateOAuthState(r, state) {
		slog.Warn("slack oauth: invalid state")
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Warn("slack oauth: no code provided")
		http.Redirect(w, r, "/settings?slack=declined", http.StatusFound)
		return
	}

	tokens, err := h.exchangeCodeForTokens(ctx, code)
	if err != nil {
		slog.Error("slack oauth: code exchange failed", "error", err)
		http.Error(w, "code exchange failed", http.StatusInternalServerError)
		return
	}

	if err := h.saveInstallation(ctx, tokens); err != nil {
		slog.Error("slack oauth: save installation failed", "error", err)
		http.Error(w, "failed to save installation", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings?slack=connected", http.StatusFound)
}

func (h *OAuthHandler) exchangeCodeForTokens(ctx context.Context, code string) (*oauthTokenResponse, error) {
	reqBody := url.Values{}
	reqBody.Set("client_id", h.clientID)
	reqBody.Set("client_secret", h.clientSecret)
	reqBody.Set("code", code)
	reqBody.Set("redirect_uri", h.redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/oauth.v2.access", bytes.NewReader([]byte(reqBody.Encode())))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp oauthTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("unmarshal token response: %w", err)
	}
	if !tokenResp.Ok {
		return nil, fmt.Errorf("slack oauth error: %s", tokenResp.Error)
	}

	return &tokenResp, nil
}

func (h *OAuthHandler) saveInstallation(ctx context.Context, tokens *oauthTokenResponse) error {
	// TODO: get the org from the request context and create an Installation record
	return nil
}

// HandleDisconnect revokes the Slack token and removes the installation.
func (h *OAuthHandler) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	w.WriteHeader(http.StatusNoContent)
}

type oauthTokenResponse struct {
	Ok          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
	AppID       string `json:"app_id"`
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	BotUserID   string `json:"bot_user_id,omitempty"`
	Team        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
}

func generateRandomState() string {
	// TODO: implement real random state generation
	return "slack_random_state"
}

func setOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:  "slack_oauth_state",
		Value: state,
	})
}

func validateOAuthState(r *http.Request, state string) bool {
	cookie, err := r.Cookie("slack_oauth_state")
	if err != nil {
		return false
	}
	return cookie.Value == state
}
