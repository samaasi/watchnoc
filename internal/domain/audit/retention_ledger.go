package audit

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// RetentionLedger records each batch of records purged by the retention worker.
// This itself is never purged — it is a permanent compliance record.
type RetentionLedger struct {
	model.AppendOnlyBase // Append-only — no update or delete ever

	OrgID uint64 `gorm:"not null;index" tenant:"org_id" hard_purge:"f" tombstone:"hard_purge" json:"org_id"`

	// TableName_ is the table from which records were purged.
	// ("deploy_events", "approvals" etc.) — named TableName_ to avoid conflict
	// with the GORM TableName() method.
	TableName_ string `gorm:"column:table_name;not null;size:64" json:"table_name"`

	// RecordsPurged is the count of records soft-deleted in this run.
	RecordsPurged int `gorm:"not null" json:"records_purged"`

	// OldestPurgedAt is the triggered_at/occurred_at of the oldest record removed.
	OldestPurgedAt time.Time `gorm:"not null" json:"oldest_purged_at"`

	// RetentionDaysApplied is the org's retention setting at the time of purge.
	RetentionDaysApplied int `gorm:"not null" json:"retention_days_applied"`
}

func (RetentionLedger) TableName() string { return "retention_ledger" }
