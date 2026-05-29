package dora

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

var (
	ErrMetricsCalculationFailed = errors.Internal("METRICS_CALCULATION_FAILED", "failed to calculate DORA metrics", nil)
)
