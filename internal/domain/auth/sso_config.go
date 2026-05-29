package auth

import (
	"github.com/samaasi/watchnoc/internal/platform/model"
	"gorm.io/datatypes"
)

// SSOProtocol identifies the enterprise SSO standard in use.
type SSOProtocol string

const (
	SSOProtocolSAML SSOProtocol = "saml2"
	SSOProtocolOIDC SSOProtocol = "oidc"
)

// SSOConfig stores the SAML 2.0 or OIDC configuration for an enterprise org.
// When present, all authentication for the org is routed through the SSO provider.
// Gate 4 only — built when an enterprise deal is blocked by its absence.
type SSOConfig struct {
	model.Base

	OrgID    uint64      `gorm:"not null;uniqueIndex" tenant:"org_id" json:"org_id"`
	Protocol SSOProtocol `gorm:"not null;size:16"     audit:"true" json:"protocol"`

	// IsEnforced means non-SSO logins are blocked for this org.
	IsEnforced bool `gorm:"not null;default:false" audit:"true" json:"is_enforced"`

	// Config holds the protocol-specific configuration as JSONB.
	// SAML: {entity_id, sso_url, x509_cert, attribute_mapping}
	// OIDC: {client_id, discovery_url, scopes, claim_mapping}
	// Encrypted at the application layer before storage.
	Config datatypes.JSON `gorm:"type:jsonb;not null" pii:"true" crypto:"true" audit:"true" json:"-"`

	// Domain is the email domain enforced for SSO matching (e.g. "acme.com").
	Domain string `gorm:"not null;size:255;index" json:"domain"`

	// SetupByUserID is the admin who completed SSO configuration.
	SetupByUserID *uint64 `json:"setup_by_user_id,omitempty"`
}

func (SSOConfig) TableName() string { return "sso_configs" }
