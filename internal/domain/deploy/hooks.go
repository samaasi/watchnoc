package deploy

import (
	"fmt"

	"gorm.io/gorm"
)

// BeforeCreate assigns the Snowflake ID if not already set,
// and derives RiskLevel from RiskScore.
func (d *DeployEvent) BeforeCreate(tx *gorm.DB) error {
	if d.ID == 0 {
		return fmt.Errorf("deploy_event: ID must be set before insert (use pkg/snowflake)")
	}
	d.RiskLevel = ToRiskLevel(d.RiskScore)
	return nil
}
