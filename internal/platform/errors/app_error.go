package errors

type AppError struct {
	Code           string           `json:"code"`
	Category       ErrorCategory    `json:"-"`
	Message        string           `json:"message"`
	UserMessage    string           `json:"user_message"`
	HTTPStatus     int              `json:"http_status"`
	Retryable      bool             `json:"-"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
	InternalErr    error            `json:"-"`
}

func (e AppError) Error() string {
	if e.InternalErr != nil {
		return e.Message + ": " + e.InternalErr.Error()
	}
	return e.Message
}

func (e AppError) Unwrap() error {
	return e.InternalErr
}
