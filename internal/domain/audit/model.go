package audit

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
	"gorm.io/datatypes"
)

// AuditEventType classifies what happened.
type AuditEventType string

const (
	EventDeployIngested     AuditEventType = "deploy.ingested"
	EventDeployVoided       AuditEventType = "deploy.voided"
	EventApprovalRequested  AuditEventType = "approval.requested"
	EventApprovalGranted    AuditEventType = "approval.granted"
	EventApprovalRejected   AuditEventType = "approval.rejected"
	EventEvidenceGenerated  AuditEventType = "evidence.generated"
	EventPolicyChanged      AuditEventType = "policy.changed"
	EventMemberAdded        AuditEventType = "org.member_added"
	EventMemberRemoved      AuditEventType = "org.member_removed"
	EventGitHubConnected    AuditEventType = "integration.github_connected"
	EventGitHubDisconnected AuditEventType = "integration.github_disconnected"
	EventSlackConnected     AuditEventType = "integration.slack_connected"
	EventAuditChainVerified AuditEventType = "audit.chain_verified"
)

// AuditRecord is the immutable, hash-chained audit log entry.
// Once written, this record must never be modified or deleted.
// The GORM BeforeCreate hook computes RecordHash and links PreviousHash.
//
// DO NOT embed model.Base — AppendOnlyBase omits UpdatedAt and DeletedAt
// to enforce the immutability contract at the schema level.
type AuditRecord struct {
	model.AppendOnlyBase

	// OrgID scopes the record to a tenant.
	OrgID uint64 `gorm:"not null;index:idx_audit_org_time" tenant:"org_id" hard_purge:"f" tombstone:"hard_purge" json:"org_id"`

	// EventType classifies the action for filtering and compliance mapping.
	EventType AuditEventType `gorm:"not null;size:64;index" json:"event_type"`

	// ActorUserID is the user who caused this event. Null for system-generated events.
	ActorUserID *uint64 `gorm:"index" json:"actor_user_id,omitempty"`

	// ActorService identifies the system component for non-human events.
	// Example: "github-webhook-processor", "approval-notifier"
	ActorService string `gorm:"size:64" json:"actor_service,omitempty"`

	// ResourceType and ResourceID identify the primary resource affected.
	// Example: ResourceType="deploy_event", ResourceID="7891234567890"
	ResourceType string `gorm:"not null;size:64" json:"resource_type"`
	ResourceID   string `gorm:"not null;size:64" json:"resource_id"`

	// Payload holds the full event data as JSONB.
	// Structured differently per EventType — see audit/schema docs.
	Payload datatypes.JSON `gorm:"type:jsonb;not null" json:"payload"`

	// --- Hash Chain ---
	// RecordHash is SHA-256(id || org_id || event_type || resource_id || payload || previous_hash).
	// Computed in the BeforeCreate GORM hook — never set manually.
	RecordHash string `gorm:"not null;size:64;uniqueIndex" hash:"sha256" json:"record_hash"`

	// PreviousHash is the RecordHash of the immediately preceding record for this org.
	// Null only for the very first record of an org.
	PreviousHash *string `gorm:"size:64" hash:"sha256" json:"previous_hash,omitempty"`

	// OccurredAt is the business time of the event (may differ from created_at
	// if events are ingested with a delay, e.g. webhook retries).
	OccurredAt time.Time `gorm:"not null;index:idx_audit_org_time" json:"occurred_at"`

	// TraceID links this record to the distributed trace that produced it.
	TraceID string `gorm:"size:64;index" evidence_export:"false" json:"trace_id,omitempty"`

	// IPAddress of the actor for security audit purposes. PII.
	IPAddress string `gorm:"size:45" pii:"true" json:"ip_address,omitempty"`
}

func (AuditRecord) TableName() string { return "audit_records" }
