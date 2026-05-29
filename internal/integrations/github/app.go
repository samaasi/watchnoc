// internal/integrations/github/app.go

package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	gogithub "github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

// AppClient handles all GitHub App authentication concerns.
// It is the single entry point for minting installation tokens and
// constructing authenticated API clients for a given installation.
type AppClient struct {
	appID      int64
	privateKey *rsa.PrivateKey
	baseURL    string

	// tokenCache stores active installation tokens to avoid minting
	// a new token on every API call. Tokens are valid for 1 hour;
	// we proactively refresh 5 minutes before expiry.
	tokenCache sync.Map // map[int64]*cachedToken  (key: installationID)
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

func (t *cachedToken) isValid() bool {
	// Refresh 5 minutes before expiry to avoid using an about-to-expire token
	return time.Now().Before(t.expiresAt.Add(-5 * time.Minute))
}

// NewAppClient parses the PEM private key and returns a configured AppClient.
// Called once in wire.go — the client is shared across all requests.
func NewAppClient(appID int64, privateKeyPEM []byte, baseURL string) (*AppClient, error) {
	key, err := parsePrivateKey(string(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("github app: parse private key: %w", err)
	}

	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	return &AppClient{
		appID:      appID,
		privateKey: key,
		baseURL:    baseURL,
	}, nil
}

// InstallationClient returns an authenticated *gogithub.Client scoped to
// a specific installation. Uses cached tokens where possible.
func (c *AppClient) InstallationClient(ctx context.Context, installationID int64) (*gogithub.Client, error) {
	token, err := c.getInstallationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(ctx, ts)

	client, err := gogithub.NewClient(httpClient).WithEnterpriseURLs(
		c.baseURL+"/",
		c.baseURL+"/",
	)
	if err != nil {
		return nil, fmt.Errorf("github: construct client: %w", err)
	}

	return client, nil
}

// getInstallationToken returns a valid installation token, using the cache
// if available or minting a new one from GitHub if not.
func (c *AppClient) getInstallationToken(ctx context.Context, installationID int64) (string, error) {
	// Check cache first
	if cached, ok := c.tokenCache.Load(installationID); ok {
		ct := cached.(*cachedToken)
		if ct.isValid() {
			return ct.token, nil
		}
	}

	// Cache miss or expired — mint a new token using the App JWT
	appJWT, err := c.generateAppJWT()
	if err != nil {
		return "", fmt.Errorf("generate app jwt: %w", err)
	}

	// Construct a temporary client authenticated as the App (not an installation)
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: appJWT})
	appHTTPClient := oauth2.NewClient(ctx, ts)

	appClient, err := gogithub.NewClient(appHTTPClient).WithEnterpriseURLs(
		c.baseURL+"/",
		c.baseURL+"/",
	)
	if err != nil {
		return "", err
	}

	// Exchange App JWT for an installation access token
	installToken, _, err := appClient.Apps.CreateInstallationToken(
		ctx,
		installationID,
		&gogithub.InstallationTokenOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("create installation token for %d: %w", installationID, err)
	}

	ct := &cachedToken{
		token:     installToken.GetToken(),
		expiresAt: installToken.GetExpiresAt().Time,
	}
	c.tokenCache.Store(installationID, ct)

	return ct.token, nil
}

// generateAppJWT creates a short-lived JWT signed with the App's private key.
// GitHub accepts App JWTs for up to 10 minutes. We generate 8-minute tokens
// to avoid clock-skew rejections.
func (c *AppClient) generateAppJWT() (string, error) {
	now := time.Now()

	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)), // backdate 60s for clock skew
		ExpiresAt: jwt.NewNumericDate(now.Add(8 * time.Minute)),
		Issuer:    fmt.Sprintf("%d", c.appID),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign app jwt: %w", err)
	}

	return signed, nil
}

func parsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format — GitHub sometimes provides this
		parsed, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key (tried PKCS1 and PKCS8): %w", err)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return rsaKey, nil
	}

	return key, nil
}
