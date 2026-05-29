package approval

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// ApprovalStatus is the decision outcome.
type ApprovalStatus string

const (
	ApprovalPending    ApprovalStatus = "pending"
	ApprovalGranted    ApprovalStatus = "granted"
	ApprovalRejected   ApprovalStatus = "rejected"
	ApprovalExpired    ApprovalStatus = "expired"    // No action taken within deadline
	ApprovalSkipped    ApprovalStatus = "skipped"    // Policy: below threshold, auto-pass
	ApprovalSuperseded ApprovalStatus = "superseded" // Voided by a newer approval request
)

// ApprovalChannel records how the approval action was taken.
type ApprovalChannel string

const (
	ChannelWebApp ApprovalChannel = "web_app" // Clicked in the dashboard
	ChannelSlack  ApprovalChannel = "slack"   // Clicked in a Slack message (Gate 2)
	ChannelAPI    ApprovalChannel = "api"     // Programmatic (future)
)

// Approval is the formal approval or rejection record for a deploy.
// This is the primary evidence artefact for SOC 2 CC8.1.
// Treat as append-only: never update a granted/rejected record.
// If a re-review is needed, void the existing record and create a new one.
type Approval struct {
	model.Base

	// OrgID for tenant-scoped queries.
	OrgID uint64 `gorm:"not null;index" tenant:"org_id" json:"org_id"`

	// DeployEventID links to the deployment being reviewed.
	DeployEventID uint64 `gorm:"not null;uniqueIndex:idx_approval_deploy_active,where:status NOT IN ('superseded')" json:"deploy_event_id"`

	// Status is the current decision state.
	Status ApprovalStatus `gorm:"not null;size:32;default:'pending';index" json:"status"`

	// RequestedAt is when the approval request was created (may differ from created_at
	// if created asynchronously by the approval-notify worker).
	RequestedAt time.Time `gorm:"not null" json:"requested_at"`

	// ExpiresAt is the deadline. Null means no expiry enforced.
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`

	// --- Decision fields (null until decided) ---

	// ApproverUserID is the DeployGuard user who made the decision.
	ApproverUserID *uint64 `gorm:"index" json:"approver_user_id,omitempty"`

	// ApproverGitHubLogin is a denormalised copy for audit exports
	// (survives user deletion, readable without a JOIN).
	// PII — identifies a person.
	ApproverGitHubLogin string `gorm:"size:255" pii:"true" json:"approver_github_login,omitempty"`

	// DecidedAt is when the Approve or Reject action was taken.
	DecidedAt *time.Time `gorm:"index" json:"decided_at,omitempty"`

	// Channel records where the decision was made.
	Channel ApprovalChannel `gorm:"size:32" json:"channel,omitempty"`

	// Comment is the optional reviewer note. Required if the policy mandates it.
	Comment string `gorm:"type:text" json:"comment,omitempty"`

	// PolicySnapshotID links to the ApprovalPolicy version that triggered this review.
	// Ensures the policy that governed the decision is permanently recorded.
	PolicySnapshotID *uint64 `json:"policy_snapshot_id,omitempty"`

	// SlackMessageTS is the Slack message timestamp used to update/delete the approval
	// notification after a decision. Not needed for audit — internal only.
	SlackMessageTS string `gorm:"size:32" evidence_export:"false" json:"-"`
	SlackChannelID string `gorm:"size:32" evidence_export:"false" json:"-"`

}

func (Approval) TableName() string { return "approvals" }
