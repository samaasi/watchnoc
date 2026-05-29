package approval

import (
	"context"

	"gorm.io/gorm"
)

// Repository defines the approval repository interface
type Repository interface {
	Create(ctx context.Context, approval *Approval) error
	FindByID(ctx context.Context, orgID, id uint64) (*Approval, error)
	FindByDeployEventID(ctx context.Context, orgID, deployEventID uint64) (*Approval, error)
	Update(ctx context.Context, approval *Approval) error
	ListByOrg(ctx context.Context, orgID uint64, limit, offset int) ([]*Approval, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new approval repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, approval *Approval) error {
	return r.db.WithContext(ctx).Create(approval).Error
}

func (r *repository) FindByID(ctx context.Context, orgID, id uint64) (*Approval, error) {
	var approval Approval
	err := r.db.WithContext(ctx).Where("org_id = ? AND id = ?", orgID, id).First(&approval).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrApprovalNotFound
		}
		return nil, err
	}
	return &approval, nil
}

func (r *repository) FindByDeployEventID(ctx context.Context, orgID, deployEventID uint64) (*Approval, error) {
	var approval Approval
	err := r.db.WithContext(ctx).Where("org_id = ? AND deploy_event_id = ?", orgID, deployEventID).First(&approval).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrApprovalNotFound
		}
		return nil, err
	}
	return &approval, nil
}

func (r *repository) Update(ctx context.Context, approval *Approval) error {
	return r.db.WithContext(ctx).Save(approval).Error
}

func (r *repository) ListByOrg(ctx context.Context, orgID uint64, limit, offset int) ([]*Approval, error) {
	var approvals []*Approval
	err := r.db.WithContext(ctx).Where("org_id = ?", orgID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&approvals).Error
	return approvals, err
}
