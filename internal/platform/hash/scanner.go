// internal/platform/hash/scanner.go

package hash

import (
    "fmt"
    "reflect"
    "sort"

    "github.com/samaasi/watchnoc/internal/platform/tags"
)

// HashFieldReport describes one hash-tagged field for the compliance audit.
type HashFieldReport struct {
    Model     string // Struct type name, e.g. "AuditRecord"
    Field     string // Go field name, e.g. "RecordHash"
    Column    string // DB column, e.g. "record_hash"
    Algorithm string // "sha256" | "argon2id"
}

// ScanHashFields returns a sorted compliance report of every hash-tagged field
// across all registered model types. Used by the /admin/security/hash-audit
// endpoint and the security runbook generator.
//
// Register model types via RegisterModel before calling.
func ScanHashFields() []HashFieldReport {
    var reports []HashFieldReport
    for _, entry := range registeredModels {
        for _, f := range tags.FieldsWithTag(entry.modelType, "hash") {
            reports = append(reports, HashFieldReport{
                Model:     entry.name,
                Field:     f.FieldName,
                Column:    f.ColumnName,
                Algorithm: f.TagValue,
            })
        }
    }
    sort.Slice(reports, func(i, j int) bool {
        return fmt.Sprintf("%s.%s", reports[i].Model, reports[i].Field) <
            fmt.Sprintf("%s.%s", reports[j].Model, reports[j].Field)
    })
    return reports
}

type modelEntry struct {
    name      string
    modelType reflect.Type
}

var registeredModels []modelEntry

// RegisterModel registers a model type for hash field scanning.
// Called from init() functions in each domain package.
//
// Example:
//
//	func init() { hash.RegisterModel("AuditRecord", reflect.TypeOf(AuditRecord{})) }
func RegisterModel(name string, t reflect.Type) {
    registeredModels = append(registeredModels, modelEntry{name: name, modelType: t})
}
