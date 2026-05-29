package deploy

import (
	"context"
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
	"gorm.io/datatypes"
)

// DeployStatus represents the lifecycle state of a deployment.
type DeployStatus string

const (
	StatusPending  DeployStatus = "pending"  // Ingested, awaiting risk scoring
	StatusApproved DeployStatus = "approved" // Approved by a human reviewer
	StatusRejected DeployStatus = "rejected" // Rejected — deploy should be rolled back
	StatusSkipped  DeployStatus = "skipped"  // Below risk threshold — auto-passed
	StatusVoided   DeployStatus = "voided"   // Marked invalid (duplicate, test event, etc.)
)

// RiskLevel is the human-readable tier derived from RiskScore.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"      // 0–39
	RiskMedium   RiskLevel = "medium"   // 40–59
	RiskHigh     RiskLevel = "high"     // 60–79
	RiskCritical RiskLevel = "critical" // 80–100
)

// DeploymentOutcome tracks the execution state of the deployment in the source system.
type DeploymentOutcome string

const (
	OutcomePending    DeploymentOutcome = "pending"
	OutcomeInProgress DeploymentOutcome = "in_progress"
	OutcomeSuccess    DeploymentOutcome = "success"
	OutcomeFailure    DeploymentOutcome = "failure"
	OutcomeError      DeploymentOutcome = "error"
	OutcomeUnknown    DeploymentOutcome = "unknown"
)

// CIStatus tracks the state of CI checks for the deployed commit.
type CIStatus string

const (
	CIPending CIStatus = "pending"
	CIPassed  CIStatus = "passed"
	CIFailed  CIStatus = "failed"
	CIUnknown CIStatus = "unknown"
)

// HealthStatus tracks the post-deploy health of the system.
type HealthStatus string

const (
	HealthHealthy  HealthStatus = "healthy"
	HealthDegraded HealthStatus = "degraded"
	HealthFailed   HealthStatus = "failed"
	HealthUnknown  HealthStatus = "unknown"
)

// DeployEvent is the core record for every production deployment.
// This table is effectively append-only — no soft deletes, only voiding.
// All audit exports and evidence packages are generated from this table.
type DeployEvent struct {
	model.Base

	// --- Tenancy ---
	OrgID uint64 `gorm:"not null;index:idx_deploy_org_time" tenant:"org_id" hard_purge:"f" json:"org_id"`

	// --- Source ---
	// Source identifies the integration that produced this event.
	// Values: "github" | "gitlab" (Gate 3) | "manual"
	Source string `gorm:"not null;size:32;default:'github'" json:"source"`

	// --- Repository ---
	RepoOwner string `gorm:"not null;size:255" json:"repo_owner"`
	RepoName  string `gorm:"not null;size:255" json:"repo_name"`

	// --- Commit ---
	CommitSHA     string `gorm:"not null;size:40"  json:"commit_sha"`
	CommitMessage string `gorm:"type:text"         json:"commit_message,omitempty"`
	Branch        string `gorm:"size:255"          json:"branch,omitempty"`
	Environment   string `gorm:"not null;size:64"  json:"environment"` // "production", "staging"

	// --- Author --- PII: identifies the deploying person
	AuthorLogin string `gorm:"not null;size:255" pii:"true" json:"author_login"`
	AuthorEmail string `gorm:"size:255"          pii:"true" json:"author_email,omitempty"`

	// --- Timing ---
	// CommittedAt is when the code was originally authored/committed.
	// Used for calculating DORA Lead Time for Changes.
	CommittedAt *time.Time `json:"committed_at,omitempty"`

	// TriggeredAt is when the deployment event actually fired in the source system.
	// This is NOT created_at — a webhook may arrive seconds later.
	TriggeredAt time.Time `gorm:"not null;index:idx_deploy_org_time" json:"triggered_at"`

	// CompletedAt is when the deployment actually finished executing.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// DurationSeconds is CompletedAt - TriggeredAt, populated when the deploy reaches a terminal state.
	DurationSeconds *int `json:"duration_seconds,omitempty"`

	// --- Risk ---
	RiskScore int       `gorm:"not null;default:0;check:risk_score_range,risk_score >= 0 AND risk_score <= 100" json:"risk_score"`
	RiskLevel RiskLevel `gorm:"not null;size:16;default:'low'" json:"risk_level"`

	// RiskFactors is a JSONB array of the individual scoring signals that fired.
	// Example: [{"factor": "production_deploy", "weight": 30}, {"factor": "off_hours", "weight": 20}]
	RiskFactors datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"risk_factors"`

	// --- Status & Outcome ---
	// Status tracks the internal DeployGuard approval workflow state.
	Status DeployStatus `gorm:"not null;size:32;default:'pending'" json:"status"`

	// DeploymentOutcome tracks the execution state from the source system (e.g. GitHub deployment_status).
	DeploymentOutcome DeploymentOutcome `gorm:"not null;size:32;default:'unknown'" json:"deployment_outcome"`

	// CIStatus tracks the state of CI checks for the commit before or during deployment.
	CIStatus   CIStatus   `gorm:"not null;size:32;default:'unknown'" json:"ci_status"`
	CIPassedAt *time.Time `json:"ci_passed_at,omitempty"`

	// HealthStatus tracks the post-deployment health (e.g., from PagerDuty correlation).
	HealthStatus    HealthStatus `gorm:"not null;size:32;default:'unknown'" json:"health_status"`
	HealthCheckedAt *time.Time   `json:"health_checked_at,omitempty"`

	// --- Diff Stats ---
	// FilesChanged, Additions, Deletions come from the GitHub commit payload.
	FilesChanged int `gorm:"default:0" json:"files_changed"`
	Additions    int `gorm:"default:0" json:"additions"`
	Deletions    int `gorm:"default:0" json:"deletions"`

	// AffectedServices is a JSONB string array parsed from changed file paths.
	// Example: ["api", "worker", "dashboard"]
	AffectedServices datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"affected_services"`

	// --- External IDs & Metadata ---
	// GitHubDeploymentID is the GitHub Deployments API integer ID, if present.
	GitHubDeploymentID *int64 `gorm:"uniqueIndex:idx_deploy_github_id_org,where:github_deployment_id IS NOT NULL" json:"github_deployment_id,omitempty"`

	// DeployLogURL is the link to the CI/CD execution logs (e.g., GitHub Actions run URL).
	DeployLogURL string `gorm:"size:1024" json:"deploy_log_url,omitempty"`

	// WorkflowName is the pipeline workflow name, used for allow-list matching.
	WorkflowName string `gorm:"size:255" json:"workflow_name,omitempty"`

	// DeployJobName is the specific job within the workflow that performed the deploy.
	DeployJobName string `gorm:"size:255" json:"deploy_job_name,omitempty"`

	// DeployJobConclusion is the conclusion of the specific deploy job.
	DeployJobConclusion string `gorm:"size:32" json:"deploy_job_conclusion,omitempty"`

	// PromotionChainID is a shared identifier across deploys of the same commit to different environments.
	PromotionChainID *string `gorm:"size:64;index" json:"promotion_chain_id,omitempty"`

	// --- Voiding & Rollbacks ---
	// VoidedAt and VoidReason replace soft-delete for this append-only table.
	VoidedAt     *time.Time `gorm:"index" json:"voided_at,omitempty" tombstone:"void"`
	VoidReason   string     `gorm:"size:255" json:"void_reason,omitempty"`
	VoidedByUser *uint64    `json:"voided_by_user,omitempty"`

	// RevertsDeployEventID points to the previous deploy event that this deploy rolls back.
	RevertsDeployEventID *uint64 `gorm:"index" json:"reverts_deploy_event_id,omitempty"`

	// --- Webhook Payload ---
	// RawPayload stores the original webhook body for reprocessing and debugging.
	// Stored compressed in practice (application code handles gzip before insert).
	RawPayload datatypes.JSON `gorm:"type:jsonb" evidence_export:"false" json:"-"`

	// --- Efficiency Tracking ---
	// CIJobDurationSeconds tracks the actual execution time of the CI pipeline.
	CIJobDurationSeconds *int `json:"ci_job_duration_seconds,omitempty"`

	// PRReviewDurationSeconds tracks the time from PR open to PR merge (review latency).
	PRReviewDurationSeconds *int `json:"pr_review_duration_seconds,omitempty"`

	// --- Incident & MTTR Tracking ---
	IncidentProvider  string `gorm:"size:32" json:"incident_provider,omitempty"` // "pagerduty", "datadog"
	LinkedIncidentID  string `gorm:"size:255;index" json:"linked_incident_id,omitempty"`
	LinkedIncidentURL string `gorm:"size:1024" json:"linked_incident_url,omitempty"`
}

func (DeployEvent) TableName() string { return "deploy_events" }

// ToRiskLevel derives the human-readable tier from the numeric score.
func ToRiskLevel(score int) RiskLevel {
	switch {
	case score >= 80:
		return RiskCritical
	case score >= 60:
		return RiskHigh
	case score >= 40:
		return RiskMedium
	default:
		return RiskLow
	}
}

// Service defines the interface for deploy domain operations.
type Service interface {
	IngestFromWebhook(ctx context.Context, req IngestRequest) (uint64, error)
	ExistsByGitHubDeploymentID(ctx context.Context, orgID uint64, deploymentID int64) (bool, error)
	UpdateDeploymentStatus(ctx context.Context, req StatusUpdateRequest) error
	UpdateCIStatus(ctx context.Context, req CIStatusUpdateRequest) error
	GetDeployForApproval(ctx context.Context, orgID, deployID uint64) (*DeployForApproval, error)
}

// IngestRequest defines the request for ingesting a new deploy event.
type IngestRequest struct {
	OrgID              uint64
	Source             string
	RepoOwner          string
	RepoName           string
	CommitSHA          string
	CommitMessage      string
	Branch             string
	Environment        string
	AuthorLogin        string
	AuthorEmail        string
	CommittedAt        *time.Time
	TriggeredAt        time.Time
	IsRevert           bool
	RawPayload         []byte
	GitHubDeploymentID *int64
	GitHubRunID        *int64
	WorkflowName       string
	DeployLogURL       string
	CompletedAt        *time.Time
	DurationSeconds    *int
}

// StatusUpdateRequest defines the request for updating deploy status.
type StatusUpdateRequest struct {
	GitHubDeploymentID int64
	RepoFullName       string
	State              string
	OccurredAt         time.Time
	LogURL             string
	CompletedAt        *time.Time
}

// CIStatusUpdateRequest defines the request for updating CI status.
type CIStatusUpdateRequest struct {
	OrgID        uint64
	RepoFullName string
	CommitSHA    string
	CIStatus     CIStatus
}
