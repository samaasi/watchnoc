package audit

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// AuditWORMManifest records each batch exported to S3 Object Lock WORM storage.
// Used by enterprise customers whose auditors require cryptographic proof of immutability
// beyond the Postgres hash chain.
type AuditWORMManifest struct {
	model.AppendOnlyBase

	OrgID uint64 `gorm:"not null;index" json:"org_id"`

	// BatchStart and BatchEnd are the audit_record IDs included in this batch.
	BatchStartID uint64 `gorm:"not null" json:"batch_start_id"`
	BatchEndID   uint64 `gorm:"not null" json:"batch_end_id"`

	// RecordCount is the number of AuditRecords in this batch.
	RecordCount int `gorm:"not null" json:"record_count"`

	// S3Bucket and S3Key identify the WORM object.
	S3Bucket string `gorm:"not null;size:255" json:"s3_bucket"`
	S3Key    string `gorm:"not null;size:1024" json:"s3_key"`

	// ManifestHash is SHA-256 of the entire batch content — independently verifiable.
	ManifestHash string `gorm:"not null;size:64;uniqueIndex" json:"manifest_hash"`

	// RetainUntil is the S3 Object Lock retention date — records cannot be deleted before this.
	RetainUntil time.Time `gorm:"not null" json:"retain_until"`
}

func (AuditWORMManifest) TableName() string { return "audit_worm_manifests" }
