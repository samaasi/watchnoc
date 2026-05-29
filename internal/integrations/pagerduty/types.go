package pagerduty

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// Installation represents a PagerDuty webhook connection for an organisation.
// PagerDuty uses a per-org webhook secret; there is no OAuth flow.
type Installation struct {
	model.Base `tombstone:"hard_purge"`

	OrgID uint64 `gorm:"not null;uniqueIndex" tenant:"org_id"`
	// WebhookSecret is the shared secret used to validate HMAC-SHA256 signatures.
	// Registered by the customer in their PagerDuty Generic Webhook V3 settings.
	WebhookSecret string `gorm:"not null;size:255" crypto:"true" mask:"partial"`
	// PagerDutyAccountID links the installation to the customer's PagerDuty subdomain.
	AccountID string `gorm:"size:64" audit:"true"`
	RevokedAt *time.Time
}

func (Installation) TableName() string { return "pagerduty_installations" }

// ServiceMap maps a PagerDuty Service name to a DeployGuard affected_service slug.
// Allows customers to reconcile naming differences (e.g. "billing-svc" vs "billing-api").
type ServiceMap struct {
	model.Base

	OrgID              uint64 `gorm:"not null;index" tenant:"org_id"`
	PagerDutyService   string `gorm:"not null;size:255"`
	DeployGuardService string `gorm:"not null;size:255"`
}

func (ServiceMap) TableName() string { return "pagerduty_service_maps" }

// PagerDutyIncident is DeployGuard's internal representation of a PagerDuty incident.
// Normalised from the V3 webhook payload — no raw PagerDuty types escape this package.
type PagerDutyIncident struct {
	ID         string
	Title      string
	Status     string // "triggered" | "acknowledged" | "resolved"
	Urgency    string // "high" | "low"
	HTMLURL    string
	CreatedAt  time.Time
	ResolvedAt *time.Time
	Service    PagerDutyService
}

type PagerDutyService struct {
	ID   string
	Name string
	URL  string
}

// HealthStatus is the health status of a deployment.
type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "healthy"
	HealthStatusDegraded HealthStatus = "degraded"
	HealthStatusRestored HealthStatus = "restored"
)

// V3WebhookPayload is the raw PagerDuty Generic V3 Webhook payload.
type V3WebhookPayload struct {
	ID        string            `json:"id"`
	Event     string            `json:"event"`
	CreatedAt time.Time         `json:"created_at"`
	Data      V3WebhookData     `json:"data"`
	Headers   map[string]string `json:"headers"`
}

type V3WebhookData struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Self       string           `json:"self"`
	HTMLURL    string           `json:"html_url"`
	Title      string           `json:"title"`
	Status     string           `json:"status"`
	Urgency    string           `json:"urgency"`
	CreatedAt  time.Time        `json:"created_at"`
	Service    V3WebhookService `json:"service"`
	ResolvedAt *time.Time       `json:"resolved_at,omitempty"`
}

type V3WebhookService struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Self string `json:"self"`
	Name string `json:"name"`
}
