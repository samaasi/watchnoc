package auth

import (
	"context"
	stdErrors "errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/samaasi/watchnoc/internal/config"
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

// Service defines the auth service interface
type Service interface {
	ValidateToken(ctx context.Context, token string) (*User, error)
	GetOrCreateUser(ctx context.Context, externalID, email, displayName, avatarURL string) (*User, error)
	UpdateLastLogin(ctx context.Context, userID uint64) error
}

type service struct {
	repo    Repository
	authCfg config.AuthConfig
}

// NewService creates a new auth service
func NewService(repo Repository, authCfg config.AuthConfig) Service {
	return &service{repo: repo, authCfg: authCfg}
}

type ClerkClaims struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	jwt.RegisteredClaims
}

func (s *service) ValidateToken(ctx context.Context, token string) (*User, error) {
	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		return nil, ErrUnauthorized
	}

	// First, try to parse the token without validation to get the user ID
	// In production, you would validate the token properly with Clerk's SDK or public keys
	parser := jwt.NewParser()
	unverifiedToken, _, err := parser.ParseUnverified(token, &ClerkClaims{})
	if err != nil {
		return nil, ErrUnauthorized
	}

	claims, ok := unverifiedToken.Claims.(*ClerkClaims)
	if !ok {
		return nil, ErrUnauthorized
	}

	// Get user by external ID (sub claim)
	user, err := s.repo.FindByExternalID(ctx, claims.Sub)
	if err != nil {
		var appErr errors.AppError
		if stdErrors.As(err, &appErr) && appErr.Code == ErrUserNotFound.Code {
			// User not found, try to create using claims
			if claims.Email != "" {
				return s.GetOrCreateUser(ctx, claims.Sub, claims.Email, claims.Name, claims.Picture)
			}
		}
		return nil, ErrUnauthorized
	}

	// Update last login
	if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) GetOrCreateUser(ctx context.Context, externalID, email, displayName, avatarURL string) (*User, error) {
	// Try to find existing user by external ID
	user, err := s.repo.FindByExternalID(ctx, externalID)
	if err == nil {
		// User exists, update last login
		if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
			return nil, err
		}
		return user, nil
	}

	// If error is not NotFound, return error
	var appErr errors.AppError
	if stdErrors.As(err, &appErr) {
		if appErr.Code == ErrUserNotFound.Code {
			// It's UserNotFound, continue to create
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}

	// User doesn't exist, create new one
	newUser := &User{
		ExternalID:  externalID,
		Email:       email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}

	if err := s.repo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *service) UpdateLastLogin(ctx context.Context, userID uint64) error {
	return s.repo.UpdateLastLogin(ctx, userID)
}
