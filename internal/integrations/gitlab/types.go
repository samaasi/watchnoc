package gitlab

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// Installation represents a GitLab OAuth connection for an organisation.
type Installation struct {
	model.Base `tombstone:"hard_purge"`

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

func (Installation) TableName() string { return "gitlab_installations" }

// RepositoryInfo is a lightweight repo descriptor returned by enrichment calls.
type RepositoryInfo struct {
	ID            int64
	Name          string
	FullName      string
	Private       bool
	DefaultBranch string
	Description   string
}

// LinkedMR holds merge request data associated with a deploy commit.
type LinkedMR struct {
	IID         int
	Title       string
	State       string // "opened" | "closed" | "merged"
	Merged      bool
	AuthorLogin string
	WebURL      string
}

// CommitInfo is the enriched commit data fetched via the GitLab API.
type CommitInfo struct {
	ID          string
	Message     string
	Author      CommitAuthor
	CommittedAt time.Time
}

type CommitAuthor struct {
	Login string
	Email string
	Name  string
}
