
package gitlab

import (
	"context"
	"sync"
	"time"
)

// Client is the authenticated GitLab API client for a specific org installation.
type Client struct {
	orgID       uint64
	accessToken string
	baseURL     string
}

// ClientFactory creates Clients for organizations.
type ClientFactory struct {
	clientID     string
	clientSecret string
	tokenStore   TokenStore
	installRepo  InstallationRepository
	baseURL      string
}

// TokenStore persists and retrieves OAuth tokens for GitLab installations.
type TokenStore interface {
	GetTokens(ctx context.Context, orgID uint64) (*OAuthTokens, error)
	SaveTokens(ctx context.Context, orgID uint64, tokens *OAuthTokens) error
}

// OAuthTokens holds the full OAuth 2.0 credential set for one installation.
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(
	clientID, clientSecret string,
	tokenStore TokenStore,
	installRepo InstallationRepository,
	baseURL string,
) *ClientFactory {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	return &ClientFactory{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenStore:   tokenStore,
		installRepo:  installRepo,
		baseURL:      baseURL,
	}
}

// ForOrg returns a Client for the given organization ID.
func (f *ClientFactory) ForOrg(ctx context.Context, orgID uint64) (*Client, error) {
	tokens, err := f.tokenStore.GetTokens(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// TODO: Check if token needs refreshing
	return &Client{
		orgID:       orgID,
		accessToken: tokens.AccessToken,
		baseURL:     f.baseURL,
	}, nil
}

// cachedToken stores active tokens to avoid unnecessary API calls.
type cachedToken struct {
	token     string
	expiresAt time.Time
}

func (t *cachedToken) isValid() bool {
	return time.Now().Before(t.expiresAt.Add(-5 * time.Minute))
}

// tokenCache stores cached tokens per org ID.
var tokenCache sync.Map // map[uint64]*cachedToken
