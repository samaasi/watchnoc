package linear

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// Installation represents a Linear OAuth connection for an organisation.
type Installation struct {
	model.Base `tombstone:"hard_purge"`

	OrgID           uint64 `gorm:"not null;uniqueIndex" tenant:"org_id"`
	LinearWorkspace string `gorm:"size:255" audit:"true"`

	// AccessToken is the long-lived OAuth token.
	// Encrypted at rest using platform/crypto.
	AccessToken string `gorm:"not null;size:512" crypto:"true" mask:"partial"`

	// WebhookSecret is used to validate incoming HMAC-SHA256 signatures.
	WebhookSecret string `gorm:"size:255" crypto:"true" mask:"partial"`
}

func (Installation) TableName() string { return "linear_installations" }

type StateType string

const (
	StateTypeBacklog   StateType = "backlog"
	StateTypeUnstarted StateType = "unstarted"
	StateTypeStarted   StateType = "started"
	StateTypeCompleted StateType = "completed"
	StateTypeCanceled  StateType = "canceled"
)

// LinearIssue represents a Linear issue with fields relevant to DeployGuard.
type LinearIssue struct {
	Key          string
	ID           string
	Title        string
	State        string
	StateType    StateType
	CreatedAt    time.Time
	UpdatedAt    time.Time
	URL          string
	AssigneeID   string
	AssigneeName string
}

// OAuthTokens holds the full OAuth 2.0 credential set for one installation.
type OAuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
}

// DeployLinkage represents the relationship between a DeployEvent and a Linear issue.
type DeployLinkage struct {
	DeployEventID   uint64
	OrgID           uint64
	LinearKey       string
	LinearURL       string
	LinearTitle     string
	LinearState     string
	LinearStateType string
	AssigneeName    string
	LinkMethod      LinkMethod
	LinkedAt        time.Time
	IssueSnapshot   *LinearIssue
}

// LinkMethod represents how the linkage was created.
type LinkMethod string

const (
	LinkMethodManual        LinkMethod = "manual"
	LinkMethodCommitMessage LinkMethod = "commit_message"
	LinkMethodBranchName    LinkMethod = "branch_name"
	LinkMethodPRTitle       LinkMethod = "pr_title"
	LinkMethodReconciler    LinkMethod = "reconciler"
)
