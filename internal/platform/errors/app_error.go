package errors

import "net/http"

type AppError struct {
	Code             string            `json:"code"`
	Category         ErrorCategory     `json:"-"`
	Message          string            `json:"message"`
	UserMessage      string            `json:"user_message"`
	HTTPStatus       int               `json:"http_status"`
	Retryable        bool              `json:"-"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
	InternalErr      error             `json:"-"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Rule    string `json:"rule,omitempty"`
}

type ErrorCategory string

const (
	CategoryValidation ErrorCategory = "validation"
	CategoryNotFound   ErrorCategory = "not_found"
	CategoryForbidden  ErrorCategory = "forbidden"
	CategoryInternal   ErrorCategory = "internal"
)

func (e AppError) Error() string {
	if e.InternalErr != nil {
		return e.Message + ": " + e.InternalErr.Error()
	}
	return e.Message
}

func (e AppError) Unwrap() error {
	return e.InternalErr
}

func NewAppError(code, message, userMessage string, httpStatus int, category ErrorCategory) AppError {
	return AppError{
		Code:        code,
		Category:    category,
		Message:     message,
		UserMessage: userMessage,
		HTTPStatus:  httpStatus,
	}
}

func NotFound(code, message string) AppError {
	return NewAppError(code, message, "The requested resource was not found", http.StatusNotFound, CategoryNotFound)
}

func Validation(code, message string, validationErrors ...ValidationError) AppError {
	err := NewAppError(code, message, "Invalid request data", http.StatusUnprocessableEntity, CategoryValidation)
	err.ValidationErrors = validationErrors
	return err
}

func Internal(code, message string, internalErr error) AppError {
	err := NewAppError(code, message, "An internal error occurred", http.StatusInternalServerError, CategoryInternal)
	err.InternalErr = internalErr
	return err
}

func Unauthorized(code, message string) AppError {
	return NewAppError(code, message, "Unauthorized", http.StatusUnauthorized, CategoryForbidden)
}

func Forbidden(code, message string) AppError {
	return NewAppError(code, message, "Forbidden", http.StatusForbidden, CategoryForbidden)
}

func Conflict(code, message string) AppError {
	return NewAppError(code, message, "Conflict", http.StatusConflict, CategoryValidation)
}
