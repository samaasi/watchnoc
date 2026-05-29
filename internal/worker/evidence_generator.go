package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/samaasi/watchnoc/internal/domain/evidence"
)

const TaskEvidenceGeneration = "evidence:generate"

type EvidenceGenerationPayload struct {
	OrgID    uint64 `json:"org_id"`
	DeployID uint64 `json:"deploy_id"`
}

type EvidenceGeneratorWorker struct {
	evidenceSvc evidence.Service
}

func NewEvidenceGeneratorWorker(evidenceSvc evidence.Service) *EvidenceGeneratorWorker {
	return &EvidenceGeneratorWorker{evidenceSvc: evidenceSvc}
}

func (w *EvidenceGeneratorWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload EvidenceGenerationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		slog.Error("evidence generator: malformed payload", "error", err)
		return nil
	}

	report, err := w.evidenceSvc.GenerateReport(ctx, payload.DeployID)
	if err != nil {
		return fmt.Errorf("evidence generation failed for deploy %d: %w", payload.DeployID, err)
	}

	slog.Info("evidence report generated",
		"deploy_id", payload.DeployID,
		"report", report,
	)
	return nil
}
