package slack

import (
	"time"

	"github.com/samaasi/watchnoc/internal/platform/model"
)

// SlackInstallation stores the bot token and workspace details for an org's
// Slack connection. The bot_token is encrypted at rest — stored as ciphertext,
// decrypted by the SlackClient at runtime using the platform key.
type SlackInstallation struct {
	model.Base

	OrgID uint64 `gorm:"not null;uniqueIndex" tenant:"org_id" json:"org_id"`

	// TeamID is the Slack workspace identifier (e.g. "T01234ABCD").
	TeamID string `gorm:"not null;size:32" json:"team_id"`

	// TeamName is the Slack workspace display name.
	TeamName string `gorm:"size:255" json:"team_name"`

	// BotUserID is the Slack bot's user ID within the workspace.
	BotUserID string `gorm:"size:32" json:"bot_user_id"`

	// BotTokenEncrypted holds the AES-256-GCM ciphertext of the bot OAuth token.
	// NEVER log or serialize this field. Decryption happens in SlackClient only.
	BotTokenEncrypted string `gorm:"not null;type:text" pii:"true" crypto:"true" mask:"partial" json:"-"`

	// DefaultChannelID is the channel where approval notifications are posted
	// when no service-specific channel is configured.
	DefaultChannelID string `gorm:"size:32" json:"default_channel_id,omitempty"`

	// RevokedAt is set when the Slack app is uninstalled from the workspace.
	RevokedAt *time.Time `json:"revoked_at,omitempty"`

}

func (SlackInstallation) TableName() string { return "slack_installations" }
