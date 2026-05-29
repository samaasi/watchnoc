package jira

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// LinkageEngine links Jira issues to DeployEvent records.
// It runs in three modes:
//  1. Manual — triggered when a user pastes a ticket URL (Gate 1)
//  2. Automatic — triggered after a deploy is ingested, scans commit message
//     and PR title for ticket keys (Gate 2)
//  3. Reconciliation — triggered nightly to catch any missed linkages (Gate 2)
type LinkageEngine struct {
	resolver      *IssueResolver
	linkRepo      LinkageRepository
	deployService interface{} // deploy.Service (stub)
}

// NewLinkageEngine creates a new LinkageEngine.
func NewLinkageEngine(resolver *IssueResolver, linkRepo LinkageRepository, deployService interface{}) *LinkageEngine {
	return &LinkageEngine{
		resolver:      resolver,
		linkRepo:      linkRepo,
		deployService: deployService,
	}
}

// LinkageRepository is the interface for storing Deploy-Jira linkages.
type LinkageRepository interface {
	Create(ctx context.Context, linkage *DeployLinkage) error
	FindByDeployEventID(ctx context.Context, deployEventID uint64) ([]*DeployLinkage, error)
	Update(ctx context.Context, linkage *DeployLinkage) error
}

// NewLinkageRepository creates a new LinkageRepository (stub).
func NewLinkageRepository(db interface{}) LinkageRepository {
	return &stubLinkageRepository{}
}

type stubLinkageRepository struct{}

func (s *stubLinkageRepository) Create(ctx context.Context, linkage *DeployLinkage) error {
	return nil
}
func (s *stubLinkageRepository) FindByDeployEventID(ctx context.Context, deployEventID uint64) ([]*DeployLinkage, error) {
	return nil, nil
}
func (s *stubLinkageRepository) Update(ctx context.Context, linkage *DeployLinkage) error {
	return nil
}

// DeployLinkage represents the relationship between a DeployEvent and a Jira issue.
type DeployLinkage struct {
	DeployEventID      uint64
	OrgID              uint64
	JiraKey            string
	JiraURL            string
	JiraSummary        string
	JiraStatus         string
	JiraStatusCategory string // "done" | "in-progress" | "to-do"
	JiraIssueType      string
	AssigneeName       string
	LinkMethod         LinkMethod // "manual" | "commit_message" | "branch_name" | "pr_title" | "reconciler"
	LinkedAt           time.Time
	IssueSnapshot      *JiraIssue // stored as JSONB — point-in-time snapshot for audit
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

// LinkManual creates a ticket linkage from a user-pasted URL or key.
// This is the Gate 1 path — no auto-detection required.
func (e *LinkageEngine) LinkManual(
	ctx context.Context,
	orgID uint64,
	deployEventID uint64,
	rawInput string,
) (*DeployLinkage, error) {
	key, err := e.resolver.ExtractKeyFromURL(rawInput)
	if err != nil {
		return nil, fmt.Errorf("invalid Jira reference %q: %w", rawInput, err)
	}

	// Resolve the issue to validate it exists and fetch its current state
	issue, err := e.resolver.ResolveIssue(ctx, orgID, key)
	if err != nil {
		if err == ErrIssueNotFound {
			return nil, fmt.Errorf("Jira issue %s not found — check the key and your Jira connection", key)
		}
		return nil, fmt.Errorf("resolve jira issue %s: %w", key, err)
	}

	linkage := &DeployLinkage{
		DeployEventID:      deployEventID,
		OrgID:              orgID,
		JiraKey:            issue.Key,
		JiraURL:            issue.URL,
		JiraSummary:        issue.Summary,
		JiraStatus:         issue.Status,
		JiraStatusCategory: issue.StatusCategory,
		JiraIssueType:      issue.IssueType,
		AssigneeName:       issue.AssigneeDisplayName,
		LinkMethod:         LinkMethodManual,
		LinkedAt:           time.Now().UTC(),
		IssueSnapshot:      issue,
	}

	if err := e.linkRepo.Create(ctx, linkage); err != nil {
		return nil, fmt.Errorf("create linkage: %w", err)
	}

	slog.Info("jira: manual ticket linkage created",
		"org_id", orgID,
		"deploy_event_id", deployEventID,
		"jira_key", key,
	)

	return linkage, nil
}
