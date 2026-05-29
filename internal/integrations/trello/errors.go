package trello

import (
	"errors"
	"fmt"
	"net/http"

	apperrors "github.com/samaasi/watchnoc/internal/platform/errors"
)

var (
	// ErrCardNotFound is returned when the Trello API returns 404 for a card lookup.
	ErrCardNotFound = errors.New("trello: card not found or not accessible")

	// ErrInstallationNotFound is returned when no Trello installation exists for an org.
	ErrInstallationNotFound = errors.New("trello: no installation found for organisation")

	// ErrInstallationRevoked is returned when the stored OAuth token has been revoked.
	ErrInstallationRevoked = errors.New("trello: oauth token revoked — re-authorisation required")

	// ErrRateLimitExceeded is returned when Trello returns 429.
	ErrRateLimitExceeded = errors.New("trello: rate limit exceeded")

	// ErrInvalidCardURL is returned when a URL cannot be parsed as a Trello card URL.
	ErrInvalidCardURL = errors.New("trello: invalid or unrecognised card url")

	// ErrWebhookRegistrationFailed is returned when the Trello API rejects a webhook
	// registration. Most commonly caused by a non-HTTPS callback URL.
	ErrWebhookRegistrationFailed = errors.New("trello: webhook registration failed")
)

// TrelloAPIError is the structured error returned by the Trello REST API.
type TrelloAPIError struct {
	StatusCode int
	Body       string
}

func (e *TrelloAPIError) Error() string {
	return fmt.Sprintf("trello api error %d: %s", e.StatusCode, e.Body)
}

func newAPIError(statusCode int, body []byte) *TrelloAPIError {
	return &TrelloAPIError{
		StatusCode: statusCode,
		Body:       string(body),
	}
}

// MapTrelloError converts a Trello API error into a platform AppError.
// Ensures Trello error types never escape the integrations package.
func MapTrelloError(err error) error {
	if err == nil {
		return nil
	}

	var trelloErr *TrelloAPIError
	if errors.As(err, &trelloErr) {
		switch trelloErr.StatusCode {
		case http.StatusNotFound:
			return apperrors.NotFound("DG-TRELLO-4040", "trello card")
		case http.StatusUnauthorized:
			appErr := apperrors.New(
				"DG-TRELLO-4010",
				"Trello token is invalid or revoked — reconnect Trello in settings",
				"Trello connection requires re-authorisation",
				apperrors.CategoryForbidden,
			)
			appErr.HTTPStatus = http.StatusUnauthorized
			return appErr
		case http.StatusForbidden:
			return apperrors.Forbidden("DG-TRELLO-4030", "trello permission denied — check token scope")
		case http.StatusTooManyRequests:
			appErr := apperrors.New(
				"DG-TRELLO-4290",
				"Trello rate limit exceeded",
				"Trello rate limit exceeded. Please try again later.",
				apperrors.CategoryInternal,
			)
			appErr.HTTPStatus = http.StatusTooManyRequests
			appErr.Retryable = true
			return appErr
		case http.StatusUnprocessableEntity:
			return apperrors.Validation("DG-TRELLO-4220", "trello rejected the request",
				apperrors.ValidationError{
					Field:   "trello_api",
					Message: trelloErr.Body,
				})
		}
	}

	// Token revocation — Trello returns "invalid token" as plain text with a 401.
	if errors.Is(err, ErrInstallationRevoked) {
		appErr := apperrors.New(
			"DG-TRELLO-4011",
			"Trello connection requires re-authorisation",
			"Trello connection requires re-authorisation",
			apperrors.CategoryForbidden,
		)
		appErr.HTTPStatus = http.StatusUnauthorized
		return appErr
	}

	return apperrors.Internal("DG-TRELLO-5000", "trello integration error", err)
}
