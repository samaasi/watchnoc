package app

import (
	"fmt"

	"gorm.io/gorm"
)

type App struct {
	DB *gorm.DB
	// Add other services, stores, etc. as we build them
}

func NewApp(db *gorm.DB) (*App, error) {
	return &App{
		DB: db,
	}, nil
}

func (a *App) Close() error {
	sqlDB, err := a.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB instance: %w", err)
	}
	return sqlDB.Close()
}
