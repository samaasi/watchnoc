package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Client is the authenticated Jira REST API client for a specific org installation.
// Each org has one Client, scoped to one Atlassian Cloud site (cloudId).
//
// Do not construct directly — use ClientFactory.ForOrg(ctx, orgID).
type Client struct {
	cloudID      string
	baseURL      string // https://api.atlassian.com/ex/jira/{cloudId}
	httpClient   *http.Client
	tokenStore   TokenStore
	orgID        uint64
	clientID     string
	clientSecret string

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// ClientFactory creates Clients for organizations.
type ClientFactory struct {
	clientID     string
	clientSecret string
	tokenStore   TokenStore
	installRepo  InstallationRepository
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(clientID, clientSecret string, tokenStore TokenStore, installRepo InstallationRepository) *ClientFactory {
	return &ClientFactory{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenStore:   tokenStore,
		installRepo:  installRepo,
	}
}

// ForOrg returns a Client for the given organization ID.
func (f *ClientFactory) ForOrg(ctx context.Context, orgID uint64) (*Client, error) {
	tokens, err := f.tokenStore.GetTokens(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get tokens for org %d: %w", orgID, err)
	}
	return &Client{
		cloudID:      tokens.CloudID,
		baseURL:      fmt.Sprintf("https://api.atlassian.com/ex/jira/%s", tokens.CloudID),
		httpClient:   &http.Client{},
		tokenStore:   f.tokenStore,
		orgID:        orgID,
		clientID:     f.clientID,
		clientSecret: f.clientSecret,
	}, nil
}

// TokenStore persists and retrieves OAuth tokens for Jira installations.
// Implemented by internal/integrations/jira/token_store.go using PostgreSQL.
type TokenStore interface {
	GetTokens(ctx context.Context, orgID uint64) (*OAuthTokens, error)
	SaveTokens(ctx context.Context, orgID uint64, tokens *OAuthTokens) error
}

// OAuthTokens holds the full OAuth 2.0 credential set for one installation.
type OAuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	CloudID      string    `json:"cloud_id"`
	CloudName    string    `json:"cloud_name"` // human-readable site name, e.g. "acme.atlassian.net"
	Scope        string    `json:"scope"`
}

// Do executes an authenticated HTTP request against the Jira REST API.
// Automatically refreshes the access token if it is expired or about to expire.
//
// path should be relative to the Jira base URL, e.g. "/rest/api/3/issue/ENG-1234"
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	token, err := c.validAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("jira client: get valid token: %w", err)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("jira client: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// validAccessToken returns a non-expired access token, refreshing it if needed.
// Uses a mutex to prevent concurrent refresh races.
func (c *Client) validAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Token is valid if it expires more than 5 minutes from now
	if c.accessToken != "" && time.Now().Before(c.expiresAt.Add(-5*time.Minute)) {
		return c.accessToken, nil
	}

	// Refresh the token
	tokens, err := c.tokenStore.GetTokens(ctx, c.orgID)
	if err != nil {
		return "", fmt.Errorf("get stored tokens: %w", err)
	}

	newTokens, err := c.refreshAccessToken(ctx, tokens.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("refresh access token: %w", err)
	}

	// Persist the new tokens
	newTokens.CloudID = tokens.CloudID
	newTokens.CloudName = tokens.CloudName
	if err := c.tokenStore.SaveTokens(ctx, c.orgID, newTokens); err != nil {
		// Non-fatal — in-memory token is still valid for this request
		slog.Warn("jira: failed to persist refreshed tokens", "org_id", c.orgID, "error", err)
	}

	c.accessToken = newTokens.AccessToken
	c.expiresAt = newTokens.ExpiresAt

	return c.accessToken, nil
}

// refreshAccessToken exchanges a refresh token for a new access token.
func (c *Client) refreshAccessToken(ctx context.Context, refreshToken string) (*OAuthTokens, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
		"refresh_token": refreshToken,
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh token failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"` // seconds
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode refresh response: %w", err)
	}

	return &OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
		Scope:        result.Scope,
	}, nil
}
