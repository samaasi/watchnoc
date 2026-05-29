package evidence

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

var (
	ErrEvidenceNotFound = errors.NotFound("EVIDENCE_NOT_FOUND", "evidence report not found")
	ErrGenerationFailed = errors.Internal("EVIDENCE_GENERATION_FAILED", "failed to generate evidence report", nil)
)
