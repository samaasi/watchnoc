package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Repository defines the auth data access interface
type Repository interface {
	FindByExternalID(ctx context.Context, externalID string) (*User, error)
	FindByID(ctx context.Context, userID uint64) (*User, error)
	Create(ctx context.Context, user *User) error
	UpdateLastLogin(ctx context.Context, userID uint64) error
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository creates a new GORM-backed auth repository
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) FindByExternalID(ctx context.Context, externalID string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) FindByID(ctx context.Context, userID uint64) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).First(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) Create(ctx context.Context, user *User) error {
	// Generate snowflake ID if not already set
	if user.ID == 0 {
		// TODO: Use actual snowflake generator (from internal/platform/snowflake)
		user.ID = 1 // temporary placeholder
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *gormRepository) UpdateLastLogin(ctx context.Context, userID uint64) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"last_login_at": &now,
		"updated_at":    now,
	}).Error
}
