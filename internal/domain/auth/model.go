package auth

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// User represents an authenticated principal within the system.
// Authentication is delegated to Clerk/Auth0 — we store only what we need
// for RBAC, audit attribution, and display.
type User struct {
	model.Base

	// ExternalID is the Clerk or Auth0 subject identifier (e.g. "user_2abc...").
	// This is the join key back to the auth provider.
	ExternalID string `gorm:"not null;uniqueIndex;size:255" json:"external_id"`

	// Email is PII. Used for display and Slack mention mapping.
	Email string `gorm:"not null;size:255" pii:"true" json:"email"`

	// DisplayName is shown in the dashboard and Slack notifications.
	// PII — could identify a person.
	DisplayName string `gorm:"size:255" pii:"true" json:"display_name"`

	// AvatarURL is the profile picture URL from the auth provider. PII.
	AvatarURL string `gorm:"size:1024" pii:"true" evidence_export:"false" json:"avatar_url,omitempty"`

	// LastLoginAt is updated on each successful authentication.
	LastLoginAt *time.Time `gorm:"index" json:"last_login_at,omitempty"`

}

func (User) TableName() string { return "users" }
