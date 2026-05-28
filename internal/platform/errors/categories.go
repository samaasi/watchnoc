package errors

type ErrorCategory string

const (
	CategoryValidation ErrorCategory = "validation"
	CategoryNotFound    ErrorCategory = "not_found"
	CategoryForbidden   ErrorCategory = "forbidden"
	CategoryInternal    ErrorCategory = "internal"
)

// ErrorMetadata holds metadata about the error category
type ErrorMetadata struct {
	HTTPStatus int
	Retryable   bool
}

var categoryMetadata = map[ErrorCategory]ErrorMetadata{
	CategoryValidation: {HTTPStatus: 422, Retryable: false},
	CategoryNotFound:   {HTTPStatus: 404, Retryable: false},
	CategoryForbidden:  {HTTPStatus: 403, Retryable: false},
	CategoryInternal:  {HTTPStatus: 500, Retryable: true},
}

func (c ErrorCategory) Metadata() ErrorMetadata {
	return categoryMetadata[c]
}
