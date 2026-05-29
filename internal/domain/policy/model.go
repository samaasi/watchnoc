package policy

import (
	"github.com/samaasi/watchnoc/internal/platform/model"
	"gorm.io/datatypes"
)

// ApprovalPolicy is a versioned set of rules that determines which deploys
// need human approval. Only one policy per org is "active" at a time.
// Previous versions are retained indefinitely for audit trail purposes.
type ApprovalPolicy struct {
	model.Base

	OrgID uint64 `gorm:"not null;index" tenant:"org_id" json:"org_id"`

	// Version is a monotonically increasing integer per org.
	Version int `gorm:"not null;default:1" json:"version"`

	// IsActive marks the currently enforced policy version.
	// Only one record per org may have IsActive=true.
	IsActive bool `gorm:"not null;default:false;index" audit:"true" json:"is_active"`

	// Rules is the complete policy definition as JSONB.
	// Schema: array of rule objects, each with condition + action.
	// See docs/approval-policy-schema.md for the full rule shape.
	Rules datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" audit:"true" json:"rules"`

	// Description is a human-readable summary of this policy version.
	Description string `gorm:"type:text" json:"description,omitempty"`

	// CreatedByUserID is the admin who saved this version.
	CreatedByUserID *uint64 `gorm:"index" json:"created_by_user_id,omitempty"`

}

func (ApprovalPolicy) TableName() string { return "approval_policies" }
