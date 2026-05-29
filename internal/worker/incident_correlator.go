package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/samaasi/watchnoc/internal/domain/incident"
)

const TaskIncidentCorrelation = "incident:correlate"

type IncidentCorrelationPayload struct {
	IncidentID uint64 `json:"incident_id"`
	ExternalID string `json:"external_id"`
}

type IncidentCorrelatorWorker struct {
	incidentSvc incident.Service
}

func NewIncidentCorrelatorWorker(incidentSvc incident.Service) *IncidentCorrelatorWorker {
	return &IncidentCorrelatorWorker{incidentSvc: incidentSvc}
}

func (w *IncidentCorrelatorWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload IncidentCorrelationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		slog.Error("incident correlator: malformed payload", "error", err)
		return nil
	}

	if err := w.incidentSvc.CorrelateIncident(ctx, payload.ExternalID); err != nil {
		return fmt.Errorf("incident correlation failed for %s: %w", payload.ExternalID, err)
	}

	slog.Info("incident correlation complete", "external_id", payload.ExternalID)
	return nil
}
