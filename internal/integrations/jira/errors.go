package jira

import "fmt"

// Jira-specific errors
var (
	// ErrIssueNotFound is returned when a Jira issue is not found.
	ErrIssueNotFound = fmt.Errorf("jira issue not found")
	// ErrOAuthInvalid is returned when OAuth credentials are invalid.
	ErrOAuthInvalid = fmt.Errorf("jira oauth credentials invalid")
	// ErrRateLimitExceeded is returned when Jira rate limit is exceeded.
	ErrRateLimitExceeded = fmt.Errorf("jira rate limit exceeded")
)
