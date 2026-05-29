package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/samaasi/watchnoc/internal/domain/dora"
)

const TaskDORACalculation = "dora:calculate"

type DORACalculationPayload struct {
	OrgID uint64 `json:"org_id,omitempty"`
}

type DORACalculatorWorker struct {
	doraSvc dora.Service
}

func NewDORACalculatorWorker(doraSvc dora.Service) *DORACalculatorWorker {
	return &DORACalculatorWorker{doraSvc: doraSvc}
}

func (w *DORACalculatorWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload DORACalculationPayload
	if len(t.Payload()) > 0 {
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			slog.Error("dora calculator: malformed payload", "error", err)
			return nil
		}
	}

	snapshot, err := w.doraSvc.CalculateMetrics(ctx, payload.OrgID)
	if err != nil {
		return fmt.Errorf("dora calculation failed for org %d: %w", payload.OrgID, err)
	}

	slog.Info("dora calculation complete",
		"org_id", payload.OrgID,
		"snapshot", snapshot,
	)
	return nil
}
