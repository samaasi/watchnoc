package approval

import (
	"context"
	"time"
)

// DeployReader is the only thing the approval domain needs to know
// about deployments. It defines the minimum contract.
// This interface is owned by approval, not by deploy.
// deploy/service.go implements it — but approval does not import deploy.
type DeployReader interface {
	GetDeployForApproval(ctx context.Context, orgID, deployID uint64) (*DeploySummary, error)
}

// DeploySummary is a flat struct containing only what approval needs.
// It is defined here in the approval package.
// It does not expose deploy.DeployEvent — that is an implementation detail
// of the deploy domain that approval has no business knowing about.
type DeploySummary struct {
	ID          uint64
	OrgID       uint64
	AuthorLogin string
	RiskScore   int
	RiskLevel   string
	Environment string
	RepoName    string
	TriggeredAt time.Time
}

// Service defines the approval service interface
type Service interface {
	RequestApproval(ctx context.Context, req ApprovalRequest) (uint64, error)
	GrantApproval(ctx context.Context, req GrantRequest) error
	RejectApproval(ctx context.Context, req RejectRequest) error
	GetApproval(ctx context.Context, orgID, approvalID uint64) (*Approval, error)
}

type ApprovalRequest struct {
	OrgID         uint64
	DeployEventID uint64
	ExpiresAt     *time.Time
}

type GrantRequest struct {
	OrgID               uint64
	ApprovalID          uint64
	ApproverUserID      uint64
	ApproverGitHubLogin string
	Comment             string
	Channel             ApprovalChannel
}

type RejectRequest struct {
	OrgID               uint64
	ApprovalID          uint64
	ApproverUserID      uint64
	ApproverGitHubLogin string
	Comment             string
	Channel             ApprovalChannel
}

type service struct {
	repo         Repository
	deployReader DeployReader
}

// NewService creates a new approval service
func NewService(repo Repository, deployReader DeployReader) Service {
	return &service{repo: repo, deployReader: deployReader}
}

func (s *service) RequestApproval(ctx context.Context, req ApprovalRequest) (uint64, error) {
	// Check if approval already exists for this deploy
	_, err := s.repo.FindByDeployEventID(ctx, req.OrgID, req.DeployEventID)
	if err == nil {
		return 0, ErrApprovalAlreadyExists
	}

	approval := &Approval{
		OrgID:         req.OrgID,
		DeployEventID: req.DeployEventID,
		Status:        ApprovalPending,
		RequestedAt:   time.Now(),
		ExpiresAt:     req.ExpiresAt,
	}

	if err := s.repo.Create(ctx, approval); err != nil {
		return 0, err
	}

	return approval.ID, nil
}

func (s *service) GrantApproval(ctx context.Context, req GrantRequest) error {
	approval, err := s.repo.FindByID(ctx, req.OrgID, req.ApprovalID)
	if err != nil {
		return err
	}

	if approval.Status != ApprovalPending {
		return ErrApprovalInvalidStatus
	}

	now := time.Now()
	approval.Status = ApprovalGranted
	approval.ApproverUserID = &req.ApproverUserID
	approval.ApproverGitHubLogin = req.ApproverGitHubLogin
	approval.DecidedAt = &now
	approval.Channel = req.Channel
	approval.Comment = req.Comment

	return s.repo.Update(ctx, approval)
}

func (s *service) RejectApproval(ctx context.Context, req RejectRequest) error {
	approval, err := s.repo.FindByID(ctx, req.OrgID, req.ApprovalID)
	if err != nil {
		return err
	}

	if approval.Status != ApprovalPending {
		return ErrApprovalInvalidStatus
	}

	now := time.Now()
	approval.Status = ApprovalRejected
	approval.ApproverUserID = &req.ApproverUserID
	approval.ApproverGitHubLogin = req.ApproverGitHubLogin
	approval.DecidedAt = &now
	approval.Channel = req.Channel
	approval.Comment = req.Comment

	return s.repo.Update(ctx, approval)
}

func (s *service) GetApproval(ctx context.Context, orgID, approvalID uint64) (*Approval, error) {
	return s.repo.FindByID(ctx, orgID, approvalID)
}
