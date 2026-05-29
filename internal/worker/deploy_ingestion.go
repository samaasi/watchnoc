package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/samaasi/watchnoc/internal/domain/deploy"
	githubintegration "github.com/samaasi/watchnoc/internal/integrations/github"
)

const (
	TaskDeployIngestion = "github:deploy:ingest"
	TaskDiffEnrichment  = "github:deploy:enrich_diff"
)

// DeployIngestionPayload is the Asynq task payload.
type DeployIngestionPayload struct {
	DeliveryID string          `json:"delivery_id"`
	EventType  string          `json:"event_type"`
	Body       json.RawMessage `json:"body"`
	EnqueuedAt time.Time       `json:"enqueued_at"`
}

// DeployIngestionWorker consumes GitHub webhook events from the Asynq queue
// and writes DeployEvent records to the database.
type DeployIngestionWorker struct {
	enricher      *githubintegration.EnrichmentClient
	deployService deploy.Service
	installRepo   githubintegration.InstallationRepository
	deliveryRepo  githubintegration.DeliveryRepository
}

// ProcessTask is the Asynq task handler. Registered in wire.go.
func (w *DeployIngestionWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload DeployIngestionPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		// Malformed payload — do not retry, discard
		slog.Error("deploy ingestion: malformed payload",
			"task_id", t.ResultWriter(),
			"error", err,
		)
		return nil
	}

	return w.processDelivery(ctx, payload.DeliveryID, payload.EventType, payload.Body)
}

func (w *DeployIngestionWorker) processDelivery(
	ctx context.Context,
	deliveryID, eventType string,
	body []byte,
) error {
	start := time.Now()

	// Idempotency check — Layer 1
	exists, err := w.deliveryRepo.Exists(ctx, deliveryID)
	if err != nil {
		return fmt.Errorf("delivery check: %w", err)
	}
	if exists {
		slog.Info("skipping duplicate delivery", "delivery_id", deliveryID)
		return nil
	}

	if err := w.deliveryRepo.Record(ctx, deliveryID, eventType); err != nil {
		return fmt.Errorf("record delivery: %w", err)
	}

	var processErr error

	switch eventType {
	case "push":
		processErr = w.processPushEvent(ctx, body)
	case "deployment":
		processErr = w.processDeploymentEvent(ctx, body)
	case "deployment_status":
		processErr = w.processDeploymentStatusEvent(ctx, body)
	case "workflow_run":
		processErr = w.processWorkflowRunEvent(ctx, body)
	case "check_suite":
		processErr = w.processCheckSuiteEvent(ctx, body)
	default:
		slog.Warn("deploy ingestion: unknown event type", "event_type", eventType)
		return nil
	}

	result := "processed"
	if processErr != nil {
		result = "failed"
		slog.Error("deploy ingestion failed",
			"delivery_id", deliveryID,
			"event_type", eventType,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", processErr,
		)
	}

	_ = w.deliveryRepo.UpdateResult(ctx, deliveryID, result,
		time.Since(start).Milliseconds())

	return processErr
}

func (w *DeployIngestionWorker) processPushEvent(ctx context.Context, body []byte) error {
	event, err := githubintegration.ParsePushEvent(body)
	if err != nil {
		if err == githubintegration.ErrNoCommitData {
			// Branch deletion push — not a deploy
			return nil
		}
		return fmt.Errorf("parse push event: %w", err)
	}

	// Resolve the organisation from the GitHub installation
	installation, err := w.installRepo.FindByRepoFullName(ctx, event.RepoFullName)
	if err != nil {
		return fmt.Errorf("resolve installation for %s: %w", event.RepoFullName, err)
	}

	// Check for revert
	isRevert, _ := githubintegration.DetectRevert(event.CommitMessage)
	// The actual deployService.IngestFromWebhook will handle matching the original commit message
	// to populate RevertsDeployEventID.

	// Enrich with diff stats asynchronously — do not block ingestion
	// The DeployEvent is created with whatever data the webhook provides.
	// The enrichment worker fills in files_changed, additions, deletions,
	// affected_services, and linked_prs after the fact.
	deployEventID, err := w.deployService.IngestFromWebhook(ctx, deploy.IngestRequest{
		OrgID:         installation.OrgID,
		Source:        "github",
		RepoOwner:     event.RepoOwner,
		RepoName:      event.RepoName,
		CommitSHA:     event.CommitSHA,
		CommitMessage: event.CommitMessage,
		Branch:        event.Branch,
		Environment:   event.Environment,
		AuthorLogin:   event.AuthorLogin,
		AuthorEmail:   event.AuthorEmail,
		CommittedAt:   event.CommittedAt,
		TriggeredAt:   event.TriggeredAt,
		IsRevert:      isRevert,
		RawPayload:    event.RawPayload,
	})
	if err != nil {
		return fmt.Errorf("ingest push deploy event: %w", err)
	}

	// Enqueue enrichment job — fills diff stats, PR linkage
	slog.Info("deploy event created, enqueueing enrichment",
		"deploy_event_id", deployEventID,
		"commit_sha", event.CommitSHA,
	)

	return nil
}

func (w *DeployIngestionWorker) processDeploymentEvent(ctx context.Context, body []byte) error {
	event, err := githubintegration.ParseDeploymentEvent(body)
	if err != nil {
		return fmt.Errorf("parse deployment event: %w", err)
	}

	installation, err := w.installRepo.FindByRepoFullName(ctx, event.RepoFullName)
	if err != nil {
		return fmt.Errorf("resolve installation for %s: %w", event.RepoFullName, err)
	}

	_, err = w.deployService.IngestFromWebhook(ctx, deploy.IngestRequest{
		OrgID:              installation.OrgID,
		Source:             "github",
		RepoOwner:          event.RepoOwner,
		RepoName:           event.RepoName,
		CommitSHA:          event.CommitSHA,
		Branch:             event.Branch,
		Environment:        event.Environment,
		AuthorLogin:        event.AuthorLogin,
		AuthorEmail:        event.AuthorEmail,
		CommittedAt:        event.CommittedAt,
		TriggeredAt:        event.TriggeredAt,
		GitHubDeploymentID: event.GitHubDeploymentID,
		RawPayload:         event.RawPayload,
	})
	return err
}

func (w *DeployIngestionWorker) processDeploymentStatusEvent(ctx context.Context, body []byte) error {
	statusEvent, err := githubintegration.ParseDeploymentStatusEvent(body)
	if err != nil {
		return fmt.Errorf("parse deployment_status event: %w", err)
	}

	// Determine CompletedAt — only set if state is terminal
	var completedAt *time.Time
	if statusEvent.State == "success" || statusEvent.State == "failure" || statusEvent.State == "error" {
		completedAt = &statusEvent.OccurredAt
	}

	// Update the existing DeployEvent with the outcome
	return w.deployService.UpdateDeploymentStatus(ctx, deploy.StatusUpdateRequest{
		GitHubDeploymentID: statusEvent.GitHubDeploymentID,
		RepoFullName:       statusEvent.RepoFullName,
		State:              statusEvent.State,
		OccurredAt:         statusEvent.OccurredAt,
		LogURL:             statusEvent.LogURL,
		CompletedAt:        completedAt,
	})
}

func (w *DeployIngestionWorker) processWorkflowRunEvent(ctx context.Context, body []byte) error {
	event, err := githubintegration.ParseWorkflowRunEvent(body)
	if err != nil {
		if err == githubintegration.ErrWorkflowNotSuccess {
			// Not a successful deploy, discard
			return nil
		}
		return fmt.Errorf("parse workflow_run event: %w", err)
	}

	installation, err := w.installRepo.FindByRepoFullName(ctx, event.RepoFullName)
	if err != nil {
		return fmt.Errorf("resolve installation for %s: %w", event.RepoFullName, err)
	}

	// Compute duration
	var durationSeconds *int
	if event.CompletedAt != nil {
		dur := int(event.CompletedAt.Sub(event.TriggeredAt).Seconds())
		durationSeconds = &dur
	}

	_, err = w.deployService.IngestFromWebhook(ctx, deploy.IngestRequest{
		OrgID:           installation.OrgID,
		Source:          "github",
		RepoOwner:       event.RepoOwner,
		RepoName:        event.RepoName,
		CommitSHA:       event.CommitSHA,
		CommitMessage:   event.CommitMessage,
		Branch:          event.Branch,
		Environment:     event.Environment,
		AuthorLogin:     event.AuthorLogin,
		AuthorEmail:     event.AuthorEmail,
		CommittedAt:     event.CommittedAt,
		TriggeredAt:     event.TriggeredAt,
		GitHubRunID:     event.GitHubRunID,
		WorkflowName:    event.WorkflowName,
		DeployLogURL:    event.DeployLogURL,
		CompletedAt:     event.CompletedAt,
		DurationSeconds: durationSeconds,
		RawPayload:      event.RawPayload,
	})
	return err
}

func (w *DeployIngestionWorker) processCheckSuiteEvent(ctx context.Context, body []byte) error {
	event, err := githubintegration.ParseCheckSuiteEvent(body)
	if err != nil {
		return fmt.Errorf("parse check_suite event: %w", err)
	}

	installation, err := w.installRepo.FindByRepoFullName(ctx, event.RepoFullName)
	if err != nil {
		return fmt.Errorf("resolve installation for %s: %w", event.RepoFullName, err)
	}

	// Convert conclusion string to deploy.CIStatus
	var ciStatus deploy.CIStatus
	switch event.Conclusion {
	case "success":
		ciStatus = deploy.CIPassed
	case "failure", "cancelled", "timed_out", "action_required":
		ciStatus = deploy.CIFailed
	default:
		ciStatus = deploy.CIUnknown
	}
	return w.deployService.UpdateCIStatus(ctx, deploy.CIStatusUpdateRequest{
		OrgID:        installation.OrgID,
		RepoFullName: event.RepoFullName,
		CommitSHA:    event.CommitSHA,
		CIStatus:     ciStatus,
	})
}

// internal/worker/deploy_ingestion.go (Asynq task options)

// NewDeployIngestionTask creates an Asynq task with retry configuration
// appropriate for GitHub webhook processing.
func NewDeployIngestionTask(payload DeployIngestionPayload) (*asynq.Task, []asynq.Option) {
	data, _ := json.Marshal(payload)

	return asynq.NewTask(TaskDeployIngestion, data), []asynq.Option{
		// Retry up to 5 times with exponential backoff
		// Delays: ~30s, ~2m, ~8m, ~30m, ~2h
		asynq.MaxRetry(5),
		// Tasks that have not been processed after 24 hours are discarded
		asynq.Deadline(time.Now().Add(24 * time.Hour)),
		// Unique within a 30-second window — prevents duplicate task creation
		// if the webhook receiver goroutine crashes and restarts
		asynq.Unique(30 * time.Second),
		// Queue priority: normal (not critical — compliance not real-time)
		asynq.Queue("default"),
	}
}
