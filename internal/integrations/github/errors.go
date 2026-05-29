// internal/integrations/github/errors.go

package github

import (
	"errors"
	"net/http"

	gogithub "github.com/google/go-github/v62/github"

	apperrors "github.com/samaasi/watchnoc/internal/platform/errors"
)

var (
	// ErrNoCommitData is returned when a push event has no head commit.
	// This happens on branch deletion pushes — not an error condition.
	ErrNoCommitData = errors.New("github: no commit data in push event")

	// ErrInstallationNotFound is returned when no installation record
	// exists for the repo that sent the webhook.
	ErrInstallationNotFound = errors.New("github: installation not found for repository")

	// ErrInstallationRevoked is returned when the App has been uninstalled
	// but a webhook arrives for that installation (race condition).
	ErrInstallationRevoked = errors.New("github: installation has been revoked")

	// ErrRateLimitExceeded is returned when GitHub returns 429.
	ErrRateLimitExceeded = errors.New("github: rate limit exceeded")

	// ErrInvalidSignature is returned when HMAC validation fails.
	// Never logged with the received signature value — timing attack risk.
	ErrInvalidSignature = errors.New("github: invalid webhook signature")
)

// MapGitHubError converts a GitHub API error into a platform AppError.
// This ensures GitHub SDK types never escape the integrations package.
func MapGitHubError(err error) error {
	if err == nil {
		return nil
	}

	var ghErr *gogithub.ErrorResponse
	if errors.As(err, &ghErr) {
		switch ghErr.Response.StatusCode {
		case http.StatusNotFound:
			return apperrors.NotFound("DG-GITHUB-4040", "github resource")
		case http.StatusUnauthorized:
			return apperrors.Unauthorized("DG-GITHUB-4010", "github authentication failed — check installation token")
		case http.StatusForbidden:
			return apperrors.Forbidden("DG-GITHUB-4030", "github permission denied — check App permissions")
		case http.StatusTooManyRequests:
			appErr := apperrors.New(
				"DG-GITHUB-4290",
				"GitHub rate limit exceeded",
				"GitHub rate limit exceeded. Please try again later.",
				apperrors.CategoryInternal,
			)
			appErr.HTTPStatus = http.StatusTooManyRequests
			appErr.Retryable = true
			return appErr
		case http.StatusUnprocessableEntity:
			return apperrors.Validation("DG-GITHUB-4220", "github rejected the request",
				apperrors.ValidationError{
					Field:   "github_api",
					Message: ghErr.Message,
				})
		}
	}

	var abuseErr *gogithub.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		appErr := apperrors.New(
			"DG-GITHUB-4291",
			"GitHub abuse rate limit triggered",
			"GitHub abuse rate limit triggered. Please try again later.",
			apperrors.CategoryInternal,
		)
		appErr.Retryable = true
		return appErr
	}

	return apperrors.Internal("DG-GITHUB-5000", "github integration error", err)
}
