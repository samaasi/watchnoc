// internal/integrations/github/types.go

package github

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// Installation represents a GitHub App installation for an organisation.
// One installation per org — enforced by unique index on org_id.
type Installation struct {
	model.Base `tombstone:"hard_purge"`

	OrgID                uint64 `gorm:"not null;uniqueIndex" tenant:"org_id"`
	InstallationID       int64  `gorm:"not null;uniqueIndex" audit:"true"`
	InstallerGitHubLogin string `gorm:"size:255" pii:"true" audit:"true"`
	AccountLogin         string `gorm:"not null;size:255" audit:"true"`
	AccountType          string `gorm:"not null;size:32"` // "Organization" | "User"
	SuspendedAt          *time.Time
	RevokedAt            *time.Time
}

func (Installation) TableName() string { return "github_installations" }

// RepositoryInfo is a lightweight repo descriptor returned by enrichment calls.
// Not stored in the database — used for the evidence scope selector UI.
type RepositoryInfo struct {
	ID            int64
	Name          string
	FullName      string
	Private       bool
	DefaultBranch string
	Language      string
	Description   string
}

// LinkedPR holds pull request data associated with a deploy commit.
// Stored as JSONB in deploy_events.raw_payload — not a separate table in Gate 1.
type LinkedPR struct {
	Number      int
	Title       string
	State       string // "open" | "closed"
	Merged      bool
	AuthorLogin string
	HTMLURL     string
}

// CommitInfo is the enriched commit data fetched via the GitHub API.
type CommitInfo struct {
	SHA         string
	Message     string
	Author      CommitAuthor
	CommittedAt time.Time
	Stats       CommitStats
}

type CommitAuthor struct {
	Login string
	Email string
	Name  string
}

type CommitStats struct {
	Additions int
	Deletions int
	Total     int
}
