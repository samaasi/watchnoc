package incident

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

var (
	ErrIncidentNotFound = errors.NotFound("INCIDENT_NOT_FOUND", "incident not found")
)
