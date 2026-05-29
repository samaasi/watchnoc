package jira

import (
	"context"
	"time"
)

// Installation represents a Jira integration installation for an organization.
type Installation struct {
	OrgID     uint64
	CloudID   string
	CloudName string
	CloudURL  string
	Scope     string
	Revoked   bool
}

// InstallationRepository is the interface for persisting and retrieving Jira installations.
type InstallationRepository interface {
	Create(ctx context.Context, install *Installation) error
	FindByOrgID(ctx context.Context, orgID uint64) (*Installation, error)
	FindByCloudID(ctx context.Context, cloudID string) (*Installation, error)
	MarkRevoked(ctx context.Context, orgID uint64) error
}

// JiraIssue represents a Jira issue with fields relevant to DeployGuard.
type JiraIssue struct {
	Key                 string
	ID                  string
	Summary             string
	IssueType           string
	Priority            string
	Status              string
	StatusCategory      string // "to-do" | "in-progress" | "done"
	CreatedAt           time.Time
	UpdatedAt           time.Time
	URL                 string
	AssigneeLogin       string
	AssigneeDisplayName string
	ReporterLogin       string
	ReporterDisplayName string
}

// JiraIssueResponse is the raw Jira REST API response for an issue.
type JiraIssueResponse struct {
	Key    string `json:"key"`
	ID     string `json:"id"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name           string `json:"name"`
			StatusCategory struct {
				Key  string `json:"key"`
				Name string `json:"name"`
			} `json:"statusCategory"`
		} `json:"status"`
		Issuetype struct {
			Name string `json:"name"`
		} `json:"issuetype"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
		Created  time.Time `json:"created"`
		Updated  time.Time `json:"updated"`
		Assignee *struct {
			AccountID   string `json:"accountId"`
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Reporter *struct {
			AccountID   string `json:"accountId"`
			DisplayName string `json:"displayName"`
		} `json:"reporter"`
	} `json:"fields"`
}

// NewInstallationRepository creates a new InstallationRepository (stub).
func NewInstallationRepository(db interface{}) InstallationRepository {
	return &stubInstallationRepository{}
}

type stubInstallationRepository struct{}

func (s *stubInstallationRepository) Create(ctx context.Context, install *Installation) error {
	return nil
}

func (s *stubInstallationRepository) FindByOrgID(ctx context.Context, orgID uint64) (*Installation, error) {
	return nil, nil
}

func (s *stubInstallationRepository) FindByCloudID(ctx context.Context, cloudID string) (*Installation, error) {
	return nil, nil
}

func (s *stubInstallationRepository) MarkRevoked(ctx context.Context, orgID uint64) error {
	return nil
}

// JobQueue is the interface for enqueuing Jira-related jobs.
type JobQueue interface {
	// Jira-specific jobs (stub for now)
}
