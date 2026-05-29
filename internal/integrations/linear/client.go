package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the authenticated Linear GraphQL client for a specific org installation.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	accessToken string
	orgID       uint64
}

// ClientFactory creates Clients for organizations.
type ClientFactory struct {
	clientID     string
	clientSecret string
	tokenStore   TokenStore
	installRepo  InstallationRepository
}

// TokenStore persists and retrieves OAuth tokens for Linear installations.
type TokenStore interface {
	GetTokens(ctx context.Context, orgID uint64) (*OAuthTokens, error)
	SaveTokens(ctx context.Context, orgID uint64, tokens *OAuthTokens) error
}

// InstallationRepository is the interface for storing and retrieving Linear installations.
type InstallationRepository interface {
	Create(ctx context.Context, install *Installation) error
	FindByOrgID(ctx context.Context, orgID uint64) (*Installation, error)
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
	install, err := f.installRepo.FindByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("find installation: %w", err)
	}

	return &Client{
		baseURL:     "https://api.linear.app/graphql",
		httpClient:  &http.Client{},
		accessToken: install.AccessToken,
		orgID:       orgID,
	}, nil
}

// Do executes an authenticated GraphQL request against the Linear API.
func (c *Client) Do(ctx context.Context, query string, variables map[string]any) (*http.Response, error) {
	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal graphql payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build graphql request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// GetIssue retrieves a Linear issue by its key (e.g., "ENG-123").
func (c *Client) GetIssue(ctx context.Context, key string) (*LinearIssue, error) {
	query := `
		query GetIssue($key: String!) {
			issue(id: $key) {
				id
				identifier
				title
				state {
					name
					type
				}
				createdAt
				updatedAt
				url
				assignee {
					id
					name
				}
			}
		}
	`

	resp, err := c.Do(ctx, query, map[string]any{"key": key})
	if err != nil {
		return nil, fmt.Errorf("get linear issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("linear api error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Issue struct {
				ID         string `json:"id"`
				Identifier string `json:"identifier"`
				Title      string `json:"title"`
				State      struct {
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"state"`
				CreatedAt time.Time `json:"createdAt"`
				UpdatedAt time.Time `json:"updatedAt"`
				URL       string    `json:"url"`
				Assignee  *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"assignee"`
			} `json:"issue"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode linear response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("linear graphql errors: %v", result.Errors)
	}

	issue := &LinearIssue{
		Key:       result.Data.Issue.Identifier,
		ID:        result.Data.Issue.ID,
		Title:     result.Data.Issue.Title,
		State:     result.Data.Issue.State.Name,
		StateType: StateType(result.Data.Issue.State.Type),
		CreatedAt: result.Data.Issue.CreatedAt,
		UpdatedAt: result.Data.Issue.UpdatedAt,
		URL:       result.Data.Issue.URL,
	}

	if result.Data.Issue.Assignee != nil {
		issue.AssigneeID = result.Data.Issue.Assignee.ID
		issue.AssigneeName = result.Data.Issue.Assignee.Name
	}

	return issue, nil
}
