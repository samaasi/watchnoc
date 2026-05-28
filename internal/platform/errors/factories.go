package errors

func New(code, message, userMessage string, category ErrorCategory) AppError {
	meta := category.Metadata()
	return AppError{
		Code:        code,
		Category:    category,
		Message:     message,
		UserMessage: userMessage,
		HTTPStatus:  meta.HTTPStatus,
		Retryable:   meta.Retryable,
	}
}

func NotFound(code, message string) AppError {
	return New(code, message, "The requested resource was not found", CategoryNotFound)
}

func Validation(code, message string, validationErrors ...ValidationError) AppError {
	err := New(code, message, "Invalid request data", CategoryValidation)
	err.ValidationErrors = validationErrors
	return err
}

func Internal(code, message string, internalErr error) AppError {
	err := New(code, message, "An internal error occurred", CategoryInternal)
	err.InternalErr = internalErr
	return err
}

func Unauthorized(code, message string) AppError {
	return New(code, message, "Unauthorized", CategoryForbidden)
}

func Forbidden(code, message string) AppError {
	return New(code, message, "Forbidden", CategoryForbidden)
}

func Conflict(code, message string) AppError {
	return New(code, message, "Conflict", CategoryValidation)
}
