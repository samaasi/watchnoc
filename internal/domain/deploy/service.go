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
		lastDeploy, err := s.repo.FindLastDeployByEnv(ctx, req.OrgID, req.RepoOwner, req.RepoName, req.Environment)
		if err == nil && lastDeploy != nil {
			deploy.RevertsDeployEventID = &lastDeploy.ID
		}
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
	deploy, err := s.repo.FindByGitHubDeploymentID(ctx, req.OrgID, req.GitHubDeploymentID)
	if err != nil {
		return err
	}

	switch req.State {
	case "success":
		deploy.DeploymentOutcome = OutcomeSuccess
	case "failure", "error":
		deploy.DeploymentOutcome = OutcomeFailure
	case "inactive":
		// Often signifies the environment was superseded
	}

	if req.CompletedAt != nil {
		deploy.CompletedAt = req.CompletedAt
	}

	return s.repo.Update(ctx, deploy)
}

func (s *service) UpdateCIStatus(ctx context.Context, req CIStatusUpdateRequest) error {
	// Splitting repo full name
	parts := strings.SplitN(req.RepoFullName, "/", 2)
	if len(parts) != 2 {
		return ErrDeployInvalidStatus
	}

	deploys, err := s.repo.FindByCommitSHA(ctx, req.OrgID, req.RepoFullName, req.CommitSHA)
	if err != nil {
		return err
	}

	for _, deploy := range deploys {
		deploy.CIStatus = req.CIStatus
		if err := s.repo.Update(ctx, deploy); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) ListByOrg(ctx context.Context, orgID uint64, limit, offset int) ([]*DeployEvent, error) {
	return s.repo.ListByOrg(ctx, orgID, limit, offset)
}

func (s *service) GetByID(ctx context.Context, orgID, deployID uint64) (*DeployEvent, error) {
	return s.repo.FindByID(ctx, orgID, deployID)
}
