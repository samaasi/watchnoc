package errors

// ValidationError represents a single validation error for a field
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Rule    string `json:"rule,omitempty"`
}
