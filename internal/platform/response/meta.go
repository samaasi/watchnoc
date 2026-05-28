package response

import "time"

type ResponseMeta struct {
	RequestID string    `json:"request_id"`
	TraceID   string    `json:"trace_id"`
	Timestamp  time.Time `json:"timestamp"`
	APIVersion string   `json:"api_version"`
}
