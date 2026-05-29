package auth

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

// Auth-related error codes
const (
	ErrCodeAuthUnauthorized = "DG-AUTH-4010"
	ErrCodeAuthForbidden    = "DG-AUTH-4030"
	ErrCodeAuthUserNotFound = "DG-AUTH-4040"
)

// Auth errors
var (
	ErrUnauthorized = errors.Unauthorized(ErrCodeAuthUnauthorized, "Unauthorized")
	ErrForbidden    = errors.Forbidden(ErrCodeAuthForbidden, "Forbidden")
	ErrUserNotFound = errors.NotFound(ErrCodeAuthUserNotFound, "User not found")
)
