package evidence

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
	"gorm.io/datatypes"
)

// ComplianceFramework identifies the regulatory standard being mapped.
type ComplianceFramework string

const (
	FrameworkSOC2     ComplianceFramework = "soc2"     // SOC 2 Type II — CC8.1
	FrameworkHIPAA    ComplianceFramework = "hipaa"    // HIPAA § 164.312
	FrameworkISO27001 ComplianceFramework = "iso27001" // Gate 3 — Annex A.12.1, A.14.2
)

// EvidenceReportStatus tracks async generation.
type EvidenceReportStatus string

const (
	EvidencePending    EvidenceReportStatus = "pending"
	EvidenceGenerating EvidenceReportStatus = "generating"
	EvidenceReady      EvidenceReportStatus = "ready"
	EvidenceFailed     EvidenceReportStatus = "failed"
)

// EvidenceReport is a generated compliance evidence package.
// Generation is asynchronous — status transitions from pending → generating → ready/failed.
// The PDF artifact is stored in S3; this record holds the metadata and access URL.
type EvidenceReport struct {
	model.Base

	OrgID uint64 `gorm:"not null;index" tenant:"org_id" hard_purge:"f" json:"org_id"`

	// Framework is the compliance standard this report targets.
	Framework ComplianceFramework `gorm:"not null;size:32" json:"framework"`

	// PeriodStart and PeriodEnd define the audit window.
	PeriodStart time.Time `gorm:"not null" json:"period_start"`
	PeriodEnd   time.Time `gorm:"not null" json:"period_end"`

	// Status of the async generation job.
	Status EvidenceReportStatus `gorm:"not null;size:32;default:'pending'" json:"status"`

	// Summary holds the computed statistics embedded in the report.
	// Stored here so the dashboard can display them without re-generating.
	// Example: {"total_deploys": 142, "approved_pct": 98.6, "exceptions": 3}
	Summary datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"summary"`

	// S3Key is the object key of the generated PDF in the evidence S3 bucket.
	// Null until generation completes.
	S3Key *string `gorm:"size:1024" evidence_export:"false" json:"-"`

	// DownloadURL is a short-lived pre-signed S3 URL. Re-generated on each GET request.
	// NOT stored — derived at request time. Omitted from DB serialization.
	DownloadURL string `gorm:"-" evidence_export:"false" json:"download_url,omitempty"`

	// GeneratedByUserID is the admin who triggered generation.
	GeneratedByUserID *uint64 `gorm:"index" json:"generated_by_user_id,omitempty"`

	// FailureReason is set if generation fails. Never surfaced to end users — internal only.
	FailureReason string `gorm:"type:text" evidence_export:"false" json:"-"`

	// ExpiresAt is when this report record (and S3 object) should be cleaned up.
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`
}

func (EvidenceReport) TableName() string { return "evidence_reports" }
