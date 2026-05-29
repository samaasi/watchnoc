// internal/platform/hash/guard.go

package hash

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"

	"github.com/samaasi/watchnoc/internal/platform/tags"
)

// GuardPlugin is a GORM plugin that verifies hash:"algo" fields contain a
// correctly formatted hash value before allowing an INSERT.
//
// It does NOT compute hashes — that is the responsibility of the service layer.
// This plugin is a safety net that prevents plaintext values from reaching the DB.
type GuardPlugin struct{}

func NewGuardPlugin() *GuardPlugin { return &GuardPlugin{} }

func (p *GuardPlugin) Name() string { return "deployguard:hash_guard" }

func (p *GuardPlugin) Initialize(db *gorm.DB) error {
	return db.Callback().Create().Before("gorm:create").
		Register("deployguard:hash_guard:check", p.check)
}

func (p *GuardPlugin) check(db *gorm.DB) {
	if db.Statement == nil || db.Statement.Model == nil {
		return
	}

	rv := reflect.ValueOf(db.Statement.Model)
	rt := rv.Type()
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		rv = rv.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return
	}

	hashFields := tags.FieldsWithTag(rt, "hash")
	for _, f := range hashFields {
		fv := rv.FieldByName(f.FieldName)
		if !fv.IsValid() || fv.Kind() != reflect.String {
			continue
		}
		value := fv.String()
		if value == "" {
			continue // Empty is allowed — pointer fields may be nil/empty
		}

		algo := strings.ToLower(f.TagValue)
		if err := validateHashFormat(value, algo); err != nil {
			db.AddError(fmt.Errorf(
				"hash guard: field %s (algo=%s): %w", f.FieldName, algo, err,
			))
			return
		}
	}
}

// validateHashFormat checks that value looks like a hash of the given algorithm.
// This is a format check only — it does not verify the hash is correct.
func validateHashFormat(value, algo string) error {
	switch algo {
	case "sha256":
		// SHA-256 hex digest: exactly 64 hex characters
		if len(value) != 64 {
			return fmt.Errorf("expected 64-char hex sha256, got len=%d", len(value))
		}
		if _, err := hex.DecodeString(value); err != nil {
			return fmt.Errorf("sha256 value is not valid hex: %w", err)
		}
	case "argon2id":
		// Argon2id PHC string format: $argon2id$v=...
		if !strings.HasPrefix(value, "$argon2id$") {
			return fmt.Errorf("argon2id value must start with $argon2id$")
		}
	default:
		return fmt.Errorf("unknown hash algorithm: %s", algo)
	}
	return nil
}
