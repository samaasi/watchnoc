package org

import (
	"context"
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
	"gorm.io/datatypes"
)

const (
	PlanStarter    = "starter"
	PlanCompliance = "compliance"
	PlanGrowth     = "growth"
	PlanEnterprise = "enterprise"
)

// Org is the top-level tenant. Every resource in the system belongs to an Org.
// Slug is immutable after creation — it appears in API paths and Slack messages.
type Org struct {
	model.Base

	// Name is the human-readable organisation name shown in the dashboard.
	Name string `gorm:"not null;size:255"              json:"name"`

	// Slug is the URL-safe identifier: lowercase, hyphens only, 3–48 chars.
	// Immutable after creation.
	Slug string `gorm:"not null;uniqueIndex;size:48"   json:"slug"`

	// Plan controls feature access and retention limits.
	// Values: "starter" | "compliance" | "growth" | "enterprise"
	Plan string `gorm:"not null;default:'starter';size:32" json:"plan"`

	// RetentionDays is the audit log retention enforced by the retention worker.
	// Starter: 90, Compliance: 365, Growth: 730, Enterprise: configurable.
	RetentionDays int `gorm:"not null;default:90" json:"retention_days"`

	// TrialEndsAt is null for paid plans, set for trial orgs.
	TrialEndsAt *time.Time `gorm:"index" json:"trial_ends_at,omitempty"`

	// BillingEmail is PII — redacted from logs.
	BillingEmail string `gorm:"size:255" pii:"true" audit:"true" json:"billing_email,omitempty"`

	// Settings holds org-level feature flags and configuration as JSONB.
	// Example: {"require_approval_comment": true, "slack_channel_id": "C123"}
	Settings datatypes.JSON `gorm:"type:jsonb;default:'{}'" audit:"true" json:"settings"`
}

func (Org) TableName() string { return "orgs" }

// OrgMemberRole is the access level within a specific org.
type OrgMemberRole string

const (
	RoleAdmin    OrgMemberRole = "admin"    // Full access: settings, policy, evidence
	RoleEngineer OrgMemberRole = "engineer" // Deploy timeline, approvals — no admin
	RoleViewer   OrgMemberRole = "viewer"   // Read-only: dashboard, audit log export
)

// OrgMember binds a User to an Org with a role.
// A user may belong to multiple orgs (rare but supported from day one).
type OrgMember struct {
	model.Base

	OrgID  uint64        `gorm:"not null;index:idx_org_members_org_user,unique" tenant:"org_id" json:"org_id"`
	UserID uint64        `gorm:"not null;index:idx_org_members_org_user,unique" json:"user_id"`
	Role   OrgMemberRole `gorm:"not null;size:32;default:'engineer'"            audit:"true" json:"role"`

	// InvitedBy is the UserID of the admin who added this member.
	// Null for the founding admin (self-signup).
	InvitedBy *uint64 `gorm:"index" json:"invited_by,omitempty"`
}

func (OrgMember) TableName() string { return "org_members" }

// ServiceMetadata maps logic service names to compliance scopes.
// Used for filtering evidence (e.g., only show deploys for HIPAA-scoped services).
type ServiceMetadata struct {
	model.Base

	OrgID       uint64 `gorm:"not null;uniqueIndex:idx_org_service"`
	ServiceName string `gorm:"not null;size:255;uniqueIndex:idx_org_service"`

	// HIPAAInScope indicates if this service processes/stores PHI.
	HIPAAInScope bool `gorm:"not null;default:false"`
}

func (ServiceMetadata) TableName() string { return "service_metadata" }

// Service defines the interface for org domain operations.
type Service interface {
	// Add org.Service methods as needed
}

// contextKey is the type for context keys to avoid collisions.
type contextKey string

// OrgIDContextKey is the key used to store org ID in context.
const OrgIDContextKey contextKey = "org_id"

// IDFromContext extracts the org ID from the context.
func IDFromContext(ctx context.Context) uint64 {
	orgID, ok := ctx.Value(OrgIDContextKey).(uint64)
	if !ok {
		return 0
	}
	return orgID
}

// WithOrgID adds the org ID to the context.
func WithOrgID(ctx context.Context, orgID uint64) context.Context {
	return context.WithValue(ctx, OrgIDContextKey, orgID)
}
