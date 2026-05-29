package evidence

import "github.com/samaasi/watchnoc/internal/platform/model"

// ISO27001Mapping links a deploy event to a specific ISO 27001 control.
// Created during evidence package generation — one record per deploy per control.
type ISO27001Mapping struct {
	model.Base

	OrgID            uint64  `gorm:"not null;index"            json:"org_id"`
	DeployEventID    uint64  `gorm:"not null;index"            json:"deploy_event_id"`
	EvidenceReportID *uint64 `gorm:"index"                json:"evidence_report_id,omitempty"`

	// ControlRef is the Annex A control identifier.
	// Examples: "A.12.1.2" (Change management), "A.14.2.2" (System change control)
	ControlRef string `gorm:"not null;size:32" json:"control_ref"`

	// ControlTitle is the human-readable control name for inclusion in exports.
	ControlTitle string `gorm:"not null;size:255" json:"control_title"`

	// EvidenceSummary is the text snippet explaining how this deploy satisfies the control.
	EvidenceSummary string `gorm:"type:text" json:"evidence_summary,omitempty"`
}

func (ISO27001Mapping) TableName() string { return "iso27001_mappings" }
