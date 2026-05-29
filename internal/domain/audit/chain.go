package audit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// BeforeCreate computes the hash chain for every AuditRecord.
// Called automatically by GORM before any INSERT on audit_records.
// The hash covers: id, org_id, event_type, resource_type, resource_id,
//
//	payload (canonical JSON), and previous_hash.
func (a *AuditRecord) BeforeCreate(tx *gorm.DB) error {
	prev, err := fetchLatestHash(tx, a.OrgID)
	if err != nil {
		return fmt.Errorf("audit chain: fetch previous hash: %w", err)
	}
	a.PreviousHash = prev
	a.RecordHash = computeRecordHash(a)
	return nil
}

func computeRecordHash(a *AuditRecord) string {
	payload, _ := json.Marshal(a.Payload)
	prev := ""
	if a.PreviousHash != nil {
		prev = *a.PreviousHash
	}
	raw := fmt.Sprintf("%d|%d|%s|%s|%s|%s|%s",
		a.ID, a.OrgID, a.EventType,
		a.ResourceType, a.ResourceID,
		string(payload), prev,
	)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

func fetchLatestHash(tx *gorm.DB, orgID uint64) (*string, error) {
	var latest AuditRecord
	result := tx.
		Select("record_hash").
		Where("org_id = ?", orgID).
		Order("created_at DESC").
		Limit(1).
		First(&latest)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // First record for this org — no previous hash
		}
		return nil, result.Error
	}
	return &latest.RecordHash, nil
}
