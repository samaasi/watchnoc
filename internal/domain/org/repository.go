package org

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repository defines org data access interface
type Repository interface {
	FindByID(ctx context.Context, orgID uint64) (*Org, error)
	FindBySlug(ctx context.Context, slug string) (*Org, error)
	Create(ctx context.Context, org *Org) error
	AddMember(ctx context.Context, member *OrgMember) error
	RemoveMember(ctx context.Context, orgID, userID uint64) error
	GetMember(ctx context.Context, orgID, userID uint64) (*OrgMember, error)
	GetMembers(ctx context.Context, orgID uint64) ([]OrgMember, error)
	UpdateSettings(ctx context.Context, orgID uint64, settings map[string]interface{}) error
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository creates new org repository
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) FindByID(ctx context.Context, orgID uint64) (*Org, error) {
	var org Org
	err := r.db.WithContext(ctx).First(&org, orgID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (r *gormRepository) FindBySlug(ctx context.Context, slug string) (*Org, error) {
	var org Org
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&org).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (r *gormRepository) Create(ctx context.Context, org *Org) error {
	if org.ID == 0 {
		// TODO: Use actual snowflake ID generator
		org.ID = 1
	}
	return r.db.WithContext(ctx).Create(org).Error
}

func (r *gormRepository) AddMember(ctx context.Context, member *OrgMember) error {
	// Check if member already exists
	var existing OrgMember
	err := r.db.WithContext(ctx).Where("org_id = ? AND user_id = ?", member.OrgID, member.UserID).First(&existing).Error
	if err == nil {
		return ErrOrgMemberAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Create new member
	if member.ID == 0 {
		// TODO: Use actual snowflake
		member.ID = 1
	}
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *gormRepository) GetMember(ctx context.Context, orgID, userID uint64) (*OrgMember, error) {
	var member OrgMember
	err := r.db.WithContext(ctx).Where("org_id = ? AND user_id = ?", orgID, userID).First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return &member, nil
}

func (r *gormRepository) GetMembers(ctx context.Context, orgID uint64) ([]OrgMember, error) {
	var members []OrgMember
	err := r.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&members).Error
	return members, err
}

func (r *gormRepository) RemoveMember(ctx context.Context, orgID, userID uint64) error {
	return r.db.WithContext(ctx).Where("org_id = ? AND user_id = ?", orgID, userID).Delete(&OrgMember{}).Error
}

func (r *gormRepository) UpdateSettings(ctx context.Context, orgID uint64, settings map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&Org{}).Where("id = ?", orgID).Update("settings", settings).Error
}
