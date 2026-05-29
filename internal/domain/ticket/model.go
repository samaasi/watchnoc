package ticket

import (
	"context"

	"github.com/samaasi/watchnoc/internal/platform/model"
	"gorm.io/datatypes"
)

// LinkedTicket associates a change management ticket with a deploy.
// Gate 1: manual URL entry only.
// Gate 2: JiraClient enriches TicketMetadata from the Jira API.
type LinkedTicket struct {
	model.Base

	OrgID         uint64 `gorm:"not null;index" json:"org_id"`
	DeployEventID uint64 `gorm:"not null;index" json:"deploy_event_id"`

	// TicketURL is the canonical URL entered by the deployer.
	// Example: "https://acme.atlassian.net/browse/ENG-1234"
	TicketURL string `gorm:"not null;size:1024" json:"ticket_url"`

	// TicketKey is the short identifier parsed from the URL.
	// Example: "ENG-1234"
	TicketKey string `gorm:"size:64;index" json:"ticket_key,omitempty"`

	// TicketSource identifies the tracker.
	// Values: "jira" | "linear" | "github_issue" | "manual"
	TicketSource string `gorm:"size:32;default:'manual'" json:"ticket_source"`

	// TicketMetadata holds fetched data from the ticket API (Gate 2+).
	// Example: {"title": "Deploy auth service v2", "status": "Done", "assignee": "alice"}
	TicketMetadata datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"ticket_metadata"`

	// LinkedByUserID is the DeployGuard user who added this link.
	LinkedByUserID *uint64 `gorm:"index" json:"linked_by_user_id,omitempty"`
}

func (LinkedTicket) TableName() string { return "linked_tickets" }

// Service defines the interface for ticket domain operations.
type Service interface {
	LinkTicket(ctx context.Context, req LinkRequest) error
}

// LinkRequest defines the request for linking a ticket to a deploy.
type LinkRequest struct {
	OrgID          uint64
	DeployEventID  uint64
	TicketURL      string
	TicketKey      string
	TicketSource   string
	TicketMetadata Metadata
}

// Metadata contains the cached ticket metadata.
type Metadata struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	BoardName   string   `json:"board_name"`
	ListName    string   `json:"list_name"`
	Labels      []string `json:"labels"`
}
