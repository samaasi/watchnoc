
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client is the authenticated Slack API client for a specific org installation.
type Client struct {
	orgID       uint64
	botToken    string
	baseURL     string
	httpClient  *http.Client
}

// ClientFactory creates Clients for organizations.
type ClientFactory struct {
	clientID     string
	clientSecret string
	tokenStore   TokenStore
	installRepo  InstallationRepository
	baseURL      string
}

// TokenStore persists and retrieves OAuth tokens for Slack installations.
type TokenStore interface {
	GetBotToken(ctx context.Context, orgID uint64) (string, error)
	SetBotToken(ctx context.Context, orgID uint64, token string) error
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(
	clientID, clientSecret string,
	tokenStore TokenStore,
	installRepo InstallationRepository,
	baseURL string,
) *ClientFactory {
	if baseURL == "" {
		baseURL = "https://slack.com/api"
	}

	return &ClientFactory{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenStore:   tokenStore,
		installRepo:  installRepo,
		baseURL:      baseURL,
	}
}

// WithToken creates a new Client bound to an access token.
func (f *ClientFactory) WithToken(ctx context.Context, orgID uint64) (*Client, error) {
	token, err := f.tokenStore.GetBotToken(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &Client{
		orgID:       orgID,
		botToken:    token,
		baseURL:     f.baseURL,
		httpClient:  &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

// postJSON performs a JSON POST to the Slack API with the bot token.
func (c *Client) postJSON(ctx context.Context, endpoint string, data interface{}) ([]byte, error) {
	reqBody, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("slack api request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read slack response: %w", err)
	}

	var okResp struct {
		Ok bool `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &okResp); err != nil {
		slog.Warn("slack api: malformed ok response", "resp_body", string(respBody))
		return nil, fmt.Errorf("unmarshal slack ok response: %w", err)
	}

	if !okResp.Ok {
		slog.Error("slack api error", "endpoint", endpoint, "error", okResp.Error)
		return nil, fmt.Errorf("slack api error: %s", okResp.Error)
	}

	return respBody, nil
}

// SendMessage sends a Slack message to a channel.
func (c *Client) SendMessage(ctx context.Context, channelID string, text string, blocks interface{}) error {
	req := map[string]interface{}{
		"channel": channelID,
		"text": text,
	}
	if blocks != nil {
		req["blocks"] = blocks
	}

	_, err := c.postJSON(ctx, "/chat.postMessage", req)
	return err
}
