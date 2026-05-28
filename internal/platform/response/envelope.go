package response

import (
	"time"
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

type ResponseMeta struct {
	RequestID string    `json:"request_id"`
	TraceID   string    `json:"trace_id"`
	Timestamp  time.Time `json:"timestamp"`
	APIVersion string   `json:"api_version"`
}

type Envelope[T any] struct {
	Success bool          `json:"success"`
	Data    T           `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
	Meta    ResponseMeta `json:"meta"`
}

type ErrorDetail struct {
	Code           string                     `json:"code"`
	Message        string                     `json:"message"`
	UserMessage    string                     `json:"user_message"`
	HTTPStatus     int                        `json:"http_status"`
	ValidationErrors []errors.ValidationError `json:"validation_errors,omitempty"`
}
