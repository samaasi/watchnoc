package dora

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// DORASnapshot is a pre-computed DORA metrics record for a given org and period.
// Computed nightly by the dora_calculator worker from deploy_events and incidents.
// Never update a snapshot — create a new one per calculation run.
type DORASnapshot struct {
	model.AppendOnlyBase // Append-only — each run produces a fresh record

	OrgID uint64 `gorm:"not null;index" tombstone:"hard_purge" json:"org_id"`

	// PeriodStart and PeriodEnd are the measurement window (typically 30 days).
	PeriodStart time.Time `gorm:"not null" json:"period_start"`
	PeriodEnd   time.Time `gorm:"not null" json:"period_end"`

	// DeploymentFrequency is deploys per day averaged over the period.
	DeploymentFrequency float64 `gorm:"not null;default:0" json:"deployment_frequency"`

	// DeploymentFrequencyTier: "elite" | "high" | "medium" | "low" per DORA benchmarks
	DeploymentFrequencyTier string `gorm:"not null;size:16" json:"deployment_frequency_tier"`

	// LeadTimeHours is average time from first commit to production deploy in hours.
	LeadTimeHours float64 `gorm:"not null;default:0" json:"lead_time_hours"`
	LeadTimeTier  string  `gorm:"not null;size:16"   json:"lead_time_tier"`

	// MTTRHours is Mean Time To Recovery in hours (time from incident to resolution).
	MTTRHours float64 `gorm:"not null;default:0" json:"mttr_hours"`
	MTTRTier  string  `gorm:"not null;size:16"   json:"mttr_tier"`

	// ChangeFailureRate is the percentage of deploys correlated with an incident.
	ChangeFailureRate float64 `gorm:"not null;default:0" json:"change_failure_rate"`
	ChangeFailureTier string  `gorm:"not null;size:16"   json:"change_failure_tier"`

	// TotalDeploys is the raw deploy count in the period (denominator for CFR).
	TotalDeploys int `gorm:"not null;default:0" json:"total_deploys"`

	// TotalIncidents is the raw incident count in the period.
	TotalIncidents int `gorm:"not null;default:0" json:"total_incidents"`
}

func (DORASnapshot) TableName() string { return "dora_snapshots" }
