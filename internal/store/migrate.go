package store

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/samaasi/watchnoc/internal/config"
)

// RunMigrations applies all pending database migrations
func RunMigrations(cfg config.DatabaseConfig, migrationsPath string) error {
	// If migrations path is empty, default to ./db/migrations
	if migrationsPath == "" {
		// Try to find migrations directory
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		migrationsPath = filepath.Join(wd, "db", "migrations")
	}

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found: %s", migrationsPath)
	}

	// Convert database URL to golang-migrate format (remove sslmode if needed)
	dsn, err := url.Parse(cfg.URL)
	if err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", filepath.ToSlash(migrationsPath)),
		dsn.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	log.Println("Starting database migrations...")

	// Apply all pending migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No database migrations to apply")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations applied successfully!")
	return nil
}

// RollbackMigrations rolls back the last migration
func RollbackMigrations(cfg config.DatabaseConfig, migrationsPath string) error {
	if migrationsPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		migrationsPath = filepath.Join(wd, "db", "migrations")
	}

	dsn, err := url.Parse(cfg.URL)
	if err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", filepath.ToSlash(migrationsPath)),
		dsn.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No rollback needed")
			return nil
		}
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	log.Println("Migration rolled back successfully")
	return nil
}
