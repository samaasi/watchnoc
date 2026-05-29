package gitlab

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// GitLabInstallation stores the GitLab OAuth credentials for an org.
// One GitLab connection per org — same constraint as GitHub.
type GitLabInstallation struct {
	model.Base

	OrgID uint64 `gorm:"not null;uniqueIndex" tenant:"org_id" json:"org_id"`

	// GitLabGroupID is the GitLab group or namespace ID.
	GitLabGroupID int64 `gorm:"not null" json:"gitlab_group_id"`

	// GitLabGroupPath is the human-readable namespace path (e.g. "acme-corp").
	GitLabGroupPath string `gorm:"not null;size:255" json:"gitlab_group_path"`

	// AccessTokenEncrypted is the AES-256-GCM ciphertext of the OAuth access token.
	AccessTokenEncrypted string `gorm:"not null;type:text" pii:"true" crypto:"true" mask:"partial" json:"-"`

	// RefreshTokenEncrypted is the ciphertext of the OAuth refresh token.
	RefreshTokenEncrypted string `gorm:"type:text" pii:"true" crypto:"true" mask:"partial" json:"-"`

	// TokenExpiresAt — GitLab tokens expire and must be refreshed.
	TokenExpiresAt *time.Time `gorm:"index" json:"token_expires_at,omitempty"`

	// WebhookSecretEncrypted — used to validate incoming GitLab webhook payloads.
	WebhookSecretEncrypted string `gorm:"not null;type:text" pii:"true" crypto:"true" mask:"partial" json:"-"`

	// RevokedAt is set when the OAuth app is uninstalled.
	RevokedAt *time.Time `json:"revoked_at,omitempty"`

}

func (GitLabInstallation) TableName() string { return "gitlab_installations" }
