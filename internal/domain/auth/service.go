package auth

import (
	"context"
)

// Service defines the auth service interface
type Service interface {
	ValidateToken(ctx context.Context, token string) (*User, error)
	GetOrCreateUser(ctx context.Context, externalID, email, displayName, avatarURL string) (*User, error)
	UpdateLastLogin(ctx context.Context, userID uint64) error
}

type service struct {
	repo Repository
}

// NewService creates a new auth service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) ValidateToken(ctx context.Context, token string) (*User, error) {
	// TODO: Integrate with Clerk/Auth0 SDK to verify token and get user claims
	// For now, placeholder implementation
	return nil, ErrUnauthorized
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
	var appErr platformErrors.AppError
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
