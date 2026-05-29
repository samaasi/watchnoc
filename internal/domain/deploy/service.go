package deploy

import (
	"context"
	"strings"
	"time"

	"gorm.io/datatypes"
)

// DeployForApproval is the deploy domain's own version of the return type.
// It is structurally identical to approval.DeploySummary but defined independently.
// This intentional duplication keeps the domains decoupled.
type DeployForApproval struct {
	ID          uint64
	OrgID       uint64
	AuthorLogin string
	RiskScore   int
	RiskLevel   string
	Environment string
	RepoName    string
	TriggeredAt time.Time
}

type service struct {
	repo Repository
}

// NewService creates a new deploy service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// GetDeployForApproval satisfies the approval.DeployReader interface.
// But deploy/service.go does not import the approval package to know this.
// The interface is structural in Go — if the method signature matches, it satisfies.
// There is no "implements" declaration needed.
func (s *service) GetDeployForApproval(ctx context.Context, orgID, deployID uint64) (*DeployForApproval, error) {
	event, err := s.repo.FindByID(ctx, orgID, deployID)
	if err != nil {
		return nil, err
	}

	// Return only what the caller needs — not the full DeployEvent.
	// This is intentional: we control what leaks across domain boundaries.
	return &DeployForApproval{
		ID:          event.ID,
		OrgID:       event.OrgID,
		AuthorLogin: event.AuthorLogin,
		RiskScore:   event.RiskScore,
		RiskLevel:   string(event.RiskLevel),
		Environment: event.Environment,
		RepoName:    event.RepoName,
		TriggeredAt: event.TriggeredAt,
	}, nil
}

func (s *service) IngestFromWebhook(ctx context.Context, req IngestRequest) (uint64, error) {
	// Check if deployment already exists
	if req.GitHubDeploymentID != nil {
		exists, err := s.repo.ExistsByGitHubDeploymentID(ctx, req.OrgID, *req.GitHubDeploymentID)
		if err != nil {
			return 0, err
		}
		if exists {
			return 0, ErrDeployAlreadyExists
		}
	}

	// Create deploy event
	deploy := &DeployEvent{
		OrgID:              req.OrgID,
		Source:             req.Source,
		RepoOwner:          req.RepoOwner,
		RepoName:           req.RepoName,
		CommitSHA:          req.CommitSHA,
		CommitMessage:      req.CommitMessage,
		Branch:             req.Branch,
		Environment:        req.Environment,
		AuthorLogin:        req.AuthorLogin,
		AuthorEmail:        req.AuthorEmail,
		CommittedAt:        req.CommittedAt,
		TriggeredAt:        req.TriggeredAt,
		CompletedAt:        req.CompletedAt,
		DurationSeconds:    req.DurationSeconds,
		Status:             StatusPending,
		DeploymentOutcome:  OutcomeUnknown,
		CIStatus:           CIUnknown,
		HealthStatus:       HealthUnknown,
		GitHubDeploymentID: req.GitHubDeploymentID,
		DeployLogURL:       req.DeployLogURL,
		WorkflowName:       req.WorkflowName,
	}

	// Store raw payload
	if req.RawPayload != nil {
		deploy.RawPayload = datatypes.JSON(req.RawPayload)
	}

	// Handle revert
	if req.IsRevert {
		// TODO: Find previous deploy and set RevertsDeployEventID
	}

	// Calculate risk score (placeholder for now)
	deploy.RiskScore = 0
	deploy.RiskLevel = RiskLow

	if err := s.repo.Create(ctx, deploy); err != nil {
		return 0, err
	}

	return deploy.ID, nil
}

func (s *service) ExistsByGitHubDeploymentID(ctx context.Context, orgID uint64, deploymentID int64) (bool, error) {
	return s.repo.ExistsByGitHubDeploymentID(ctx, orgID, deploymentID)
}

func (s *service) UpdateDeploymentStatus(ctx context.Context, req StatusUpdateRequest) error {
	// Split repo full name
	parts := strings.SplitN(req.RepoFullName, "/", 2)
	if len(parts) != 2 {
		return ErrDeployInvalidStatus
	}

	// Find deploy by GitHub deployment ID
	// Note: We need orgID here - in real implementation, we'd get it from repo mapping
	// For now, we'll skip orgID check (placeholder)
	// Placeholder: We need a way to find orgID from repo full name
	// For now, let's just return not found
	return ErrDeployNotFound
}

func (s *service) UpdateCIStatus(ctx context.Context, req CIStatusUpdateRequest) error {
	// TODO: Implement CI status update
	return nil
}
