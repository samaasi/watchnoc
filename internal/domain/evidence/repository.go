package evidence

import (
	"context"
	"gorm.io/gorm"
)

type Repository interface {
	Save(ctx context.Context, evidence *EvidenceReport) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Save(ctx context.Context, evidence *EvidenceReport) error {
	return r.db.WithContext(ctx).Save(evidence).Error
}
