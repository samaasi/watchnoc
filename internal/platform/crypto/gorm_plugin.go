// internal/platform/crypto/gorm_plugin.go

package crypto

import (
    "fmt"
    "reflect"
    "strings"

    "gorm.io/gorm"

    "github.com/samaasi/watchnoc/internal/platform/tags"
)

// CryptoPlugin is a GORM plugin that transparently encrypts and decrypts
// fields tagged crypto:"true". Register once at application startup.
//
//   db.Use(crypto.NewPlugin(cipher))
type CryptoPlugin struct {
    cipher *Cipher
}

func NewPlugin(cipher *Cipher) *CryptoPlugin {
    return &CryptoPlugin{cipher: cipher}
}

func (p *CryptoPlugin) Name() string { return "deployguard:crypto" }

func (p *CryptoPlugin) Initialize(db *gorm.DB) error {
    // Encrypt before any INSERT or UPDATE
    if err := db.Callback().Create().Before("gorm:create").
        Register("deployguard:crypto:encrypt_create", p.encrypt); err != nil {
        return fmt.Errorf("crypto plugin: register create callback: %w", err)
    }
    if err := db.Callback().Update().Before("gorm:update").
        Register("deployguard:crypto:encrypt_update", p.encrypt); err != nil {
        return fmt.Errorf("crypto plugin: register update callback: %w", err)
    }

    // Decrypt after any SELECT
    if err := db.Callback().Query().After("gorm:query").
        Register("deployguard:crypto:decrypt_query", p.decrypt); err != nil {
        return fmt.Errorf("crypto plugin: register query callback: %w", err)
    }
    if err := db.Callback().Row().After("gorm:row").
        Register("deployguard:crypto:decrypt_row", p.decrypt); err != nil {
        return fmt.Errorf("crypto plugin: register row callback: %w", err)
    }

    return nil
}

// encrypt walks the model's crypto-tagged string fields and replaces plaintext
// with ciphertext before the DB write.
func (p *CryptoPlugin) encrypt(db *gorm.DB) {
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

    cryptoFields := tags.FieldsWithTag(rt, "crypto")
    for _, f := range cryptoFields {
        if f.TagValue != "true" {
            continue
        }
        fv := rv.FieldByName(f.FieldName)
        if !fv.IsValid() || fv.Kind() != reflect.String || !fv.CanSet() {
            continue
        }
        plaintext := fv.String()
        if plaintext == "" {
            continue
        }
        // Skip values that are already ciphertext (re-save without modification)
        if isBase64Ciphertext(plaintext) {
            continue
        }
        ciphertext, err := p.cipher.Encrypt(plaintext)
        if err != nil {
            db.AddError(fmt.Errorf("crypto: encrypt field %s: %w", f.FieldName, err))
            return
        }
        fv.SetString(ciphertext)
    }
}

// decrypt walks the model's crypto-tagged string fields and replaces ciphertext
// with plaintext after the DB read.
func (p *CryptoPlugin) decrypt(db *gorm.DB) {
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

    cryptoFields := tags.FieldsWithTag(rt, "crypto")
    for _, f := range cryptoFields {
        if f.TagValue != "true" {
            continue
        }
        fv := rv.FieldByName(f.FieldName)
        if !fv.IsValid() || fv.Kind() != reflect.String || !fv.CanSet() {
            continue
        }
        ciphertext := fv.String()
        if ciphertext == "" {
            continue
        }
        plaintext, err := p.cipher.Decrypt(ciphertext)
        if err != nil {
            // Do not propagate decryption failures silently — surface to caller
            db.AddError(fmt.Errorf("crypto: decrypt field %s: %w", f.FieldName, err))
            return
        }
        fv.SetString(plaintext)
    }
}

// isBase64Ciphertext checks if a string begins with the crypto prefix.
// This prevents double-encryption on model re-save when the field hasn't changed,
// and is fully deterministic, eliminating the risk of misidentifying plaintext.
func isBase64Ciphertext(s string) bool {
    return strings.HasPrefix(s, "ENC[v1]")
}
