package policy

import (
	"context"
	"gorm.io/gorm"
)

type Repository interface {
	Save(ctx context.Context, p *ApprovalPolicy) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Save(ctx context.Context, p *ApprovalPolicy) error {
	return r.db.WithContext(ctx).Save(p).Error
}
