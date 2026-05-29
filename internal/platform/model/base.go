package model

import (
	"time"

	"gorm.io/gorm"
)

var idGenerator interface{ NextID() uint64 }

func SetIDGenerator(generator interface{ NextID() uint64 }) {
	idGenerator = generator
}

// Base is embedded by every non-audit model.
// It provides a Snowflake primary key and standard timestamps.
// Do NOT embed this in AuditRecord — audit records have no UpdatedAt and no DeletedAt.
type Base struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement:false" json:"id"`
	CreatedAt time.Time      `gorm:"not null;index"                 json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null"                       json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                            json:"-"`
}

// AppendOnlyBase is embedded by audit_records.
// No UpdatedAt, no DeletedAt — the record is immutable after insert.
type AppendOnlyBase struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement:false" json:"id"`
	CreatedAt time.Time `gorm:"not null;index" json:"created_at"`
}

// BeforeCreate generates a snowflake ID if not already set
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == 0 && idGenerator != nil {
		b.ID = idGenerator.NextID()
	}
	return nil
}

// BeforeCreate generates a snowflake ID for append-only base
func (b *AppendOnlyBase) BeforeCreate(tx *gorm.DB) error {
	if b.ID == 0 && idGenerator != nil {
		b.ID = idGenerator.NextID()
	}
	return nil
}
