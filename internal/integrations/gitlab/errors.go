package gitlab

import (
	"errors"

	apperrors "github.com/samaasi/watchnoc/internal/platform/errors"
)

var (
	// ErrNoCommitData is returned when a push event has no head commit.
	ErrNoCommitData = errors.New("gitlab: no commit data in push event")

	// ErrInstallationNotFound is returned when no installation record exists.
	ErrInstallationNotFound = errors.New("gitlab: installation not found")

	// ErrInstallationRevoked is returned when the app has been uninstalled.
	ErrInstallationRevoked = errors.New("gitlab: installation has been revoked")

	// ErrRateLimitExceeded is returned when GitLab returns 429.
	ErrRateLimitExceeded = errors.New("gitlab: rate limit exceeded")

	// ErrInvalidSignature is returned when webhook signature validation fails.
	ErrInvalidSignature = errors.New("gitlab: invalid webhook signature")
)

// MapGitLabError converts a GitLab API error into a platform AppError.
func MapGitLabError(err error) error {
	if err == nil {
		return nil
	}

	// TODO: Add handling for gitlab client errors when we import the client

	return apperrors.Internal("DG-GITLAB-5000", "gitlab integration error", err)
}
