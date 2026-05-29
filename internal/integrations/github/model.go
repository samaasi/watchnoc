package github

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// GitHubInstallation records the GitHub App installation for an org.
// The installation_id is the GitHub App installation handle — used to
// mint short-lived installation access tokens for API calls.
type GitHubInstallation struct {
	model.Base

	OrgID uint64 `gorm:"not null;uniqueIndex" tenant:"org_id" json:"org_id"`

	// InstallationID is the GitHub App installation integer ID.
	InstallationID int64 `gorm:"not null;uniqueIndex" json:"installation_id"`

	// InstallerGitHubLogin is the GitHub username who installed the app.
	InstallerGitHubLogin string `gorm:"size:255" pii:"true" json:"installer_github_login"`

	// AccountLogin is the GitHub org or user account that owns the installation.
	AccountLogin string `gorm:"not null;size:255" json:"account_login"`

	// AccountType: "Organization" or "User"
	AccountType string `gorm:"not null;size:32" json:"account_type"`

	// SuspendedAt is non-null if the GitHub App installation has been suspended.
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`

	// RevokedAt is set when the user uninstalls the app.
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

func (GitHubInstallation) TableName() string { return "github_installations" }
