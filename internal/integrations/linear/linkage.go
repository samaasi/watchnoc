package linear

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// LinkageEngine links Linear issues to DeployEvent records.
type LinkageEngine struct {
	resolver      *IssueResolver
	linkRepo      LinkageRepository
	deployService interface{}
}

// LinkageRepository is the interface for storing Deploy-Linear linkages.
type LinkageRepository interface {
	Create(ctx context.Context, linkage *DeployLinkage) error
	FindByDeployEventID(ctx context.Context, deployEventID uint64) ([]*DeployLinkage, error)
	Update(ctx context.Context, linkage *DeployLinkage) error
}

// NewLinkageEngine creates a new LinkageEngine.
func NewLinkageEngine(resolver *IssueResolver, linkRepo LinkageRepository, deployService interface{}) *LinkageEngine {
	return &LinkageEngine{
		resolver:      resolver,
		linkRepo:      linkRepo,
		deployService: deployService,
	}
}

// LinkManual creates a ticket linkage from a user-pasted URL or key.
func (e *LinkageEngine) LinkManual(
	ctx context.Context,
	orgID uint64,
	deployEventID uint64,
	rawInput string,
) (*DeployLinkage, error) {
	key, err := e.resolver.ExtractKeyFromURL(rawInput)
	if err != nil {
		return nil, fmt.Errorf("invalid linear reference: %w", err)
	}

	issue, err := e.resolver.ResolveIssue(ctx, orgID, key)
	if err != nil {
		return nil, fmt.Errorf("resolve linear issue: %w", err)
	}

	linkage := &DeployLinkage{
		DeployEventID:   deployEventID,
		OrgID:           orgID,
		LinearKey:       issue.Key,
		LinearURL:       issue.URL,
		LinearTitle:     issue.Title,
		LinearState:     issue.State,
		LinearStateType: string(issue.StateType),
		AssigneeName:    issue.AssigneeName,
		LinkMethod:      LinkMethodManual,
		LinkedAt:        time.Now().UTC(),
		IssueSnapshot:   issue,
	}

	if err := e.linkRepo.Create(ctx, linkage); err != nil {
		return nil, fmt.Errorf("create linkage: %w", err)
	}

	slog.Info("linear: manual ticket linkage created",
		"org_id", orgID,
		"deploy_event_id", deployEventID,
		"linear_key", key,
	)

	return linkage, nil
}
