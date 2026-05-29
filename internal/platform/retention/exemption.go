// internal/platform/retention/exemption.go

package retention

import (
    "fmt"
    "reflect"
    "strings"

    "github.com/samaasi/watchnoc/internal/platform/tags"
)

// ExemptionRegistry holds the set of table names that are exempt from the
// 365-day hard purge. Populated at startup by scanning hard_purge:"f" tags.
type ExemptionRegistry struct {
    exemptTables map[string]struct{}
}

// NewExemptionRegistry builds the registry by scanning the provided model types.
// Call once at application startup and inject into the retention worker.
//
// Example:
//
//	registry := retention.NewExemptionRegistry(
//	    retention.ModelEntry{Table: "audit_records",    Type: reflect.TypeOf(audit.AuditRecord{})},
//	    retention.ModelEntry{Table: "evidence_reports", Type: reflect.TypeOf(evidence.EvidenceReport{})},
//	    retention.ModelEntry{Table: "retention_ledger", Type: reflect.TypeOf(audit.RetentionLedger{})},
//	)
func NewExemptionRegistry(entries ...ModelEntry) *ExemptionRegistry {
    exempt := make(map[string]struct{})
    for _, entry := range entries {
        if isHardPurgeExempt(entry.Type) {
            exempt[entry.Table] = struct{}{}
        }
    }
    return &ExemptionRegistry{exemptTables: exempt}
}

// ModelEntry pairs a table name with its GORM model type for scanning.
type ModelEntry struct {
    Table string
    Type  reflect.Type
}

// IsExempt reports whether the given table name is exempt from hard purge.
// The retention worker calls this before issuing any DELETE statement.
func (r *ExemptionRegistry) IsExempt(tableName string) bool {
    _, exempt := r.exemptTables[tableName]
    return exempt
}

// ExemptTables returns all exempt table names — used for operator dashboards
// and runbook generation.
func (r *ExemptionRegistry) ExemptTables() []string {
    tables := make([]string, 0, len(r.exemptTables))
    for t := range r.exemptTables {
        tables = append(tables, t)
    }
    return tables
}

// isHardPurgeExempt returns true if any field in the struct (or any embedded
// struct) carries hard_purge:"f".
func isHardPurgeExempt(t reflect.Type) bool {
    for t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    fields := tags.FieldsWithTag(t, "hard_purge")
    for _, f := range fields {
        if strings.ToLower(f.TagValue) == "f" || strings.ToLower(f.TagValue) == "false" {
            return true
        }
    }
    return false
}

// Validate checks that no exempt table is being passed to a hard-purge operation.
// Returns an error with the table name if the operation would be illegal.
func (r *ExemptionRegistry) Validate(tableName string) error {
    if r.IsExempt(tableName) {
        return fmt.Errorf(
            "retention: hard purge is prohibited for table %q (hard_purge:\"f\" — 7-year retention required)",
            tableName,
        )
    }
    return nil
}
