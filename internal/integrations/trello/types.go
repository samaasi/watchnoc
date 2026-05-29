package trello

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// Installation represents a Trello OAuth connection for an organisation.
// One installation per org — enforced by unique index on org_id.
type Installation struct {
	model.Base `tombstone:"hard_purge"`

	OrgID            uint64 `gorm:"not null;uniqueIndex" tenant:"org_id"`
	TrelloMemberID   string `gorm:"not null;size:32" audit:"true"`
	TrelloMemberName string `gorm:"size:255" pii:"true" audit:"true"`
	TrelloUsername   string `gorm:"size:255" pii:"true" audit:"true"`
	// AccessToken is the long-lived OAuth token.
	// Encrypted at rest using platform/crypto — never stored in plaintext.
	AccessToken string `gorm:"not null;size:512" crypto:"true" mask:"partial"`
	// WebhookPathToken is a random secret embedded in the webhook URL.
	// See Section 16 for the security rationale.
	WebhookPathToken string `gorm:"not null;size:64;uniqueIndex" crypto:"true" mask:"partial"`
	RevokedAt        *time.Time
}

func (Installation) TableName() string { return "trello_installations" }

// WebhookRecord tracks Trello webhooks registered on behalf of an organisation.
// One record per board — we register at the board level, not per card.
type WebhookRecord struct {
	model.Base `tombstone:"hard_purge"`

	OrgID           uint64 `gorm:"not null;index" tenant:"org_id"`
	TrelloWebhookID string `gorm:"not null;uniqueIndex;size:32"`
	BoardID         string `gorm:"not null;size:32"`
	BoardName       string `gorm:"size:255"`
}

func (WebhookRecord) TableName() string { return "trello_webhooks" }

// CardMetadataUpdate carries changes to apply to a linked ticket's cached metadata.
// Used by the event router to update the ticket domain without importing Trello types.
type CardMetadataUpdate struct {
	CardShortID string
	NewTitle    string
	NewDesc     string
	Deleted     bool
}
