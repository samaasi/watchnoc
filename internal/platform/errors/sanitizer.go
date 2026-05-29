package errors

import (
	"strings"
)

// SanitizeError returns a safe error string suitable for the client.
// It strips sensitive information like database query details or file paths.
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	// Redact DB details if present
	if strings.Contains(msg, "sql:") || strings.Contains(msg, "gorm:") || strings.Contains(msg, "pq:") {
		return "a database error occurred"
	}

	return msg
}
