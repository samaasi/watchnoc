package deploy

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// Repository defines the deploy repository interface
type Repository interface {
	Create(ctx context.Context, deploy *DeployEvent) error
	FindByID(ctx context.Context, orgID, id uint64) (*DeployEvent, error)
	FindByGitHubDeploymentID(ctx context.Context, orgID uint64, deploymentID int64) (*DeployEvent, error)
	ExistsByGitHubDeploymentID(ctx context.Context, orgID uint64, deploymentID int64) (bool, error)
	Update(ctx context.Context, deploy *DeployEvent) error
	ListByOrg(ctx context.Context, orgID uint64, limit, offset int) ([]*DeployEvent, error)
	FindLastDeployByEnv(ctx context.Context, orgID uint64, repoOwner, repoName, environment string) (*DeployEvent, error)
	FindByCommitSHA(ctx context.Context, orgID uint64, repoFullName, commitSHA string) ([]*DeployEvent, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new deploy repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, deploy *DeployEvent) error {
	return r.db.WithContext(ctx).Create(deploy).Error
}

func (r *repository) FindByID(ctx context.Context, orgID, id uint64) (*DeployEvent, error) {
	var deploy DeployEvent
	err := r.db.WithContext(ctx).Where("org_id = ? AND id = ? AND voided_at IS NULL", orgID, id).First(&deploy).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDeployNotFound
		}
		return nil, err
	}
	return &deploy, nil
}

func (r *repository) FindByGitHubDeploymentID(ctx context.Context, orgID uint64, deploymentID int64) (*DeployEvent, error) {
	var deploy DeployEvent
	err := r.db.WithContext(ctx).Where("org_id = ? AND github_deployment_id = ? AND voided_at IS NULL", orgID, deploymentID).First(&deploy).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDeployNotFound
		}
		return nil, err
	}
	return &deploy, nil
}

func (r *repository) ExistsByGitHubDeploymentID(ctx context.Context, orgID uint64, deploymentID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&DeployEvent{}).Where("org_id = ? AND github_deployment_id = ? AND voided_at IS NULL", orgID, deploymentID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) Update(ctx context.Context, deploy *DeployEvent) error {
	return r.db.WithContext(ctx).Save(deploy).Error
}

func (r *repository) ListByOrg(ctx context.Context, orgID uint64, limit, offset int) ([]*DeployEvent, error) {
	var deploys []*DeployEvent
	err := r.db.WithContext(ctx).Where("org_id = ? AND voided_at IS NULL", orgID).
		Order("triggered_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&deploys).Error
	return deploys, err
}

func (r *repository) FindLastDeployByEnv(ctx context.Context, orgID uint64, repoOwner, repoName, environment string) (*DeployEvent, error) {
	var deploy DeployEvent
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND repo_owner = ? AND repo_name = ? AND environment = ? AND voided_at IS NULL", orgID, repoOwner, repoName, environment).
		Order("triggered_at DESC").
		First(&deploy).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDeployNotFound
		}
		return nil, err
	}
	return &deploy, nil
}

func (r *repository) FindByCommitSHA(ctx context.Context, orgID uint64, repoFullName, commitSHA string) ([]*DeployEvent, error) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return nil, nil // Invalid full name, handle gracefully or return error
	}
	repoOwner, repoName := parts[0], parts[1]

	var deploys []*DeployEvent
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND repo_owner = ? AND repo_name = ? AND commit_sha = ? AND voided_at IS NULL", orgID, repoOwner, repoName, commitSHA).
		Find(&deploys).Error
	return deploys, err
}
