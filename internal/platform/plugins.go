// internal/platform/plugins.go

package platform

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/samaasi/watchnoc/internal/platform/audit"
	"github.com/samaasi/watchnoc/internal/platform/crypto"
	"github.com/samaasi/watchnoc/internal/platform/hash"
)

// PluginConfig holds the dependencies required by domain tag plugins.
type PluginConfig struct {
	// EncryptionKey is a 32-byte AES-256 key loaded from AWS Secrets Manager
	// or the PLATFORM_ENCRYPTION_KEY environment variable.
	EncryptionKey []byte

	// AuditWriter is the audit domain service that writes field delta records.
	AuditWriter audit.AuditWriter

	// OrgIDExtractor resolves the org_id from a GORM model for audit attribution.
	OrgIDExtractor audit.OrgIDExtractor
}

// RegisterAllPlugins registers every domain-tag-driven GORM plugin on the given DB.
// Call once immediately after opening the GORM connection, before any queries run.
//
// Usage:
//
//	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
//	// ...
//	if err := platform.RegisterAllPlugins(db, cfg); err != nil {
//	    log.Fatal(err)
//	}
func RegisterAllPlugins(db *gorm.DB, cfg PluginConfig) error {
	// crypto:"true" — AES-256-GCM transparent encryption
	cipher, err := crypto.NewCipher(cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("platform: init crypto cipher: %w", err)
	}
	if err := db.Use(crypto.NewPlugin(cipher)); err != nil {
		return fmt.Errorf("platform: register crypto plugin: %w", err)
	}

	// hash:"algo" — format guard for hash fields before insert
	if err := db.Use(hash.NewGuardPlugin()); err != nil {
		return fmt.Errorf("platform: register hash guard plugin: %w", err)
	}

	// audit:"true" — field delta logger
	deltaPlugin := audit.NewDeltaPlugin(cfg.AuditWriter, cfg.OrgIDExtractor)
	if err := db.Use(deltaPlugin); err != nil {
		return fmt.Errorf("platform: register audit delta plugin: %w", err)
	}

	return nil
}
