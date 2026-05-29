package slack

import (
	"errors"

	apperrors "github.com/samaasi/watchnoc/internal/platform/errors"
)

var (
	// ErrInstallationNotFound is returned when no installation record exists for an org.
	ErrInstallationNotFound = errors.New("slack: installation not found")
	// ErrInstallationRevoked is returned when the Slack app has been uninstalled.
	ErrInstallationRevoked = errors.New("slack: installation has been revoked")
	// ErrInvalidSignature is returned when webhook signature validation fails.
	ErrInvalidSignature = errors.New("slack: invalid webhook signature")
)

// MapSlackError converts a Slack API error into a platform error.
func MapSlackError(err error) error {
	if err == nil {
		return nil
	}
	return apperrors.Internal("DG-SLACK-5000", "slack integration error", err)
}
