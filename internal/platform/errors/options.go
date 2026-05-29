package errors

// Option defines a functional option for configuring an AppError
type Option func(*AppError)

// WithMessage overrides the default message
func WithMessage(msg string) Option {
	return func(e *AppError) {
		e.Message = msg
	}
}

// WithValidationErrors adds validation errors to the AppError
func WithValidationErrors(validationErrors []ValidationError) Option {
	return func(e *AppError) {
		e.ValidationErrors = validationErrors
	}
}

// WithInternalErr sets the underlying error cause
func WithInternalErr(err error) Option {
	return func(e *AppError) {
		e.InternalErr = err
	}
}
