package policy

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

var (
	ErrPolicyNotFound = errors.NotFound("POLICY_NOT_FOUND", "approval policy not found")
	ErrEvaluationFailed = errors.Internal("POLICY_EVALUATION_FAILED", "failed to evaluate policy", nil)
)
