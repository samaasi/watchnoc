package errors

// Standard error types for the application.
var (
	ErrNotFound       = NotFound("NOT_FOUND", "resource not found")
	ErrUnauthorized   = Unauthorized("UNAUTHORIZED", "unauthorized access")
	ErrForbidden      = Forbidden("FORBIDDEN", "forbidden")
	ErrInternalServer = Internal("INTERNAL_ERROR", "internal server error", nil)
	ErrConflict       = Conflict("CONFLICT", "resource conflict")
)
