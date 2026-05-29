package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client wraps the Trello REST API v1.
// It is stateless — the access token is set per-request via WithToken().
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// tokenClient is a Client with an access token bound for a specific installation.
type tokenClient struct {
	*Client
	token string
}

func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.trello.com/1"
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// WithToken binds an access token to a client for a single installation's requests.
func (c *Client) WithToken(token string) *tokenClient {
	return &tokenClient{Client: c, token: token}
}

func (c *tokenClient) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("key", c.apiKey)
	params.Set("token", c.token)

	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("construct trello request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trello api request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read trello response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, newAPIError(resp.StatusCode, body)
	}

	return body, nil
}

// rawCard is the Trello API card shape — private to this package.
type rawCard struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Desc        string     `json:"desc"`
	ShortLink   string     `json:"shortLink"`
	ShortURL    string     `json:"shortUrl"`
	URL         string     `json:"url"`
	IDBoard     string     `json:"idBoard"`
	IDList      string     `json:"idList"`
	Due         *time.Time `json:"due"`
	DueComplete bool       `json:"dueComplete"`
	Closed      bool       `json:"closed"`
	Labels      []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
	Members []struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		FullName string `json:"fullName"`
	} `json:"members"`
}

func (c *tokenClient) GetCard(ctx context.Context, shortIDOrID string) (*rawCard, error) {
	body, err := c.get(ctx, fmt.Sprintf("/cards/%s", shortIDOrID), url.Values{
		"members": {"true"},
		"labels":  {"true"},
		"fields":  {"name,desc,shortLink,shortUrl,url,idBoard,idList,due,dueComplete,closed,labels"},
	})
	if err != nil {
		return nil, err
	}

	var card rawCard
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, fmt.Errorf("unmarshal card response: %w", err)
	}

	return &card, nil
}

func (c *tokenClient) GetBoard(ctx context.Context, boardID string) (*struct{ Name string }, error) {
	body, err := c.get(ctx, fmt.Sprintf("/boards/%s", boardID), url.Values{
		"fields": {"name"},
	})
	if err != nil {
		return nil, err
	}

	var board struct {
		Name string
	}
	if err := json.Unmarshal(body, &board); err != nil {
		return nil, fmt.Errorf("unmarshal board response: %w", err)
	}

	return &board, nil
}

func (c *tokenClient) GetList(ctx context.Context, listID string) (*struct{ Name string }, error) {
	body, err := c.get(ctx, fmt.Sprintf("/lists/%s", listID), url.Values{
		"fields": {"name"},
	})
	if err != nil {
		return nil, err
	}

	var list struct {
		Name string
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("unmarshal list response: %w", err)
	}

	return &list, nil
}

// MemberInfo is the authenticated member's profile.
type MemberInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
}

func (c *tokenClient) GetMe(ctx context.Context) (*MemberInfo, error) {
	body, err := c.get(ctx, "/members/me", url.Values{
		"fields": {"id,username,fullName"},
	})
	if err != nil {
		return nil, err
	}

	var member MemberInfo
	if err := json.Unmarshal(body, &member); err != nil {
		return nil, fmt.Errorf("unmarshal member response: %w", err)
	}

	return &member, nil
}

// CreateWebhookRequest is the payload for the Trello webhook registration API.
type CreateWebhookRequest struct {
	CallbackURL string
	IDModel     string // board ID, card ID, or member ID
	Description string
	Active      bool
}

func (c *tokenClient) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (string, error) {
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("token", c.token)
	params.Set("callbackURL", req.CallbackURL)
	params.Set("idModel", req.IDModel)
	params.Set("description", req.Description)
	if req.Active {
		params.Set("active", "true")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/webhooks?%s", c.baseURL, params.Encode()), nil)
	if err != nil {
		return "", fmt.Errorf("construct webhook request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("create trello webhook: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode >= 400 {
		return "", newAPIError(resp.StatusCode, body)
	}

	var webhook struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &webhook); err != nil {
		return "", fmt.Errorf("unmarshal webhook response: %w", err)
	}

	return webhook.ID, nil
}

func (c *tokenClient) DeleteWebhook(ctx context.Context, webhookID string) error {
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("token", c.token)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/webhooks/%s?%s", c.baseURL, webhookID, params.Encode()), nil)
	if err != nil {
		return fmt.Errorf("construct delete webhook request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete trello webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return newAPIError(resp.StatusCode, body)
	}

	return nil
}

// checkRateLimit inspects Trello's rate limit headers.
// Trello returns X-RateLimit-Interval-Ms and X-RateLimit-Limit on responses.
// If the limit is close to exhausted, we backoff proactively.
func checkRateLimit(resp *http.Response) (shouldWait bool, wait time.Duration) {
	if resp == nil {
		return false, 0
	}

	remaining := resp.Header.Get("X-RateLimit-Remaining")
	if remaining == "" {
		return false, 0
	}

	rem, err := strconv.Atoi(remaining)
	if err != nil || rem > 10 {
		return false, 0
	}

	// Under 10 requests remaining — wait 10 seconds (one rate limit window)
	slog.Warn("trello api: rate limit approaching",
		"remaining", rem,
	)
	return true, 10 * time.Second
}

// withRetry wraps a Trello API call with exponential backoff on 429 responses.
// Trello returns 429 with a Retry-After header when the rate limit is exceeded.
func withRetry(ctx context.Context, maxAttempts int, fn func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := fn()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Parse Retry-After header — Trello specifies seconds to wait
		retryAfter := resp.Header.Get("Retry-After")
		wait := 10 * time.Second // default
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			wait = time.Duration(seconds) * time.Second
		}

		slog.Warn("trello api: rate limited",
			"attempt", attempt+1,
			"retry_after", wait,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}

		lastErr = fmt.Errorf("trello rate limited after %d attempts", attempt+1)
	}

	return nil, lastErr
}
