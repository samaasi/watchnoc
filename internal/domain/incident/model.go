package incident

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
	"gorm.io/datatypes"
)

// Incident records a PagerDuty incident received via webhook.
type Incident struct {
	model.Base

	OrgID uint64 `gorm:"not null;index" json:"org_id"`

	// ExternalID is the PagerDuty incident ID. Unique per org.
	ExternalID string `gorm:"not null;size:64;index" json:"external_id"`

	// Source identifies the alerting system.
	// Values: "pagerduty" — expandable in Gate 3 (OpsGenie, VictorOps)
	Source string `gorm:"not null;size:32;default:'pagerduty'" json:"source"`

	// Title is the incident summary from the alert payload.
	Title string `gorm:"not null;size:512" json:"title"`

	// Severity: "critical" | "error" | "warning" | "info"
	Severity string `gorm:"size:32" json:"severity,omitempty"`

	// FiredAt is when PagerDuty triggered the incident.
	FiredAt time.Time `gorm:"not null;index" json:"fired_at"`

	// ResolvedAt is when the incident was marked resolved. Null if still open.
	ResolvedAt *time.Time `gorm:"index" json:"resolved_at,omitempty"`

	// RawPayload stores the original PagerDuty webhook body.
	RawPayload datatypes.JSON `gorm:"type:jsonb" json:"-"`

	// CorrelatedDeploys — populated after correlation worker runs.
	CorrelatedDeploys []IncidentDeployCorrelation `gorm:"foreignKey:IncidentID" json:"correlated_deploys,omitempty"`
}

func (Incident) TableName() string { return "incidents" }

// IncidentDeployCorrelation links an incident to a potentially causative deploy.
// Created by the incident_correlator worker — not a definitive root cause,
// but a signal surfaced to the engineer and recorded in audit evidence.
type IncidentDeployCorrelation struct {
	model.Base

	IncidentID    uint64 `gorm:"not null;index:idx_incident_deploy_corr,unique" json:"incident_id"`
	DeployEventID uint64 `gorm:"not null;index:idx_incident_deploy_corr,unique" json:"deploy_event_id"`

	// TimeDeltaSeconds is the gap in seconds between the deploy's triggered_at
	// and the incident's fired_at. Negative values mean the deploy came AFTER
	// the incident — unusual but possible with delayed webhooks.
	TimeDeltaSeconds int `gorm:"not null" json:"time_delta_seconds"`

	// CorrelationConfidence: "high" | "medium" | "low"
	// Derived from time delta: <5min=high, <15min=medium, <30min=low.
	CorrelationConfidence string `gorm:"not null;size:16;default:'low'" json:"correlation_confidence"`

}

func (IncidentDeployCorrelation) TableName() string { return "incident_deploy_correlations" }
