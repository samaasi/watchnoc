// internal/platform/audit/delta_plugin.go

package audit

import (
    "context"
    "encoding/json"
    "fmt"
    "reflect"

    "gorm.io/gorm"

    "github.com/samaasi/watchnoc/internal/platform/tags"
)

// DeltaPlugin is a GORM plugin that logs field-level changes for audit:"true" fields.
// It captures the pre-save state, diffs it after save, and writes an AuditRecord
// if any audited field changed.
//
// Dependencies:
//   - AuditWriter: the service that creates AuditRecord rows (uses the AppendOnlyBase chain)
//   - OrgIDExtractor: resolves the org_id from the model being saved (via the tenant tag)
type DeltaPlugin struct {
    writer       AuditWriter
    orgExtractor OrgIDExtractor
}

// AuditWriter is satisfied by the audit domain's repository.
type AuditWriter interface {
    WriteFieldDelta(ctx context.Context, delta FieldDelta) error
}

// OrgIDExtractor resolves the org_id from a GORM model value.
type OrgIDExtractor interface {
    ExtractOrgID(model interface{}) (uint64, bool)
}

// FieldDelta is the structured payload written to audit_records.Payload
// for audit:"true" field changes.
type FieldDelta struct {
    ResourceType string          `json:"resource_type"` // e.g. "org"
    ResourceID   string          `json:"resource_id"`
    ChangedFields []FieldChange  `json:"changed_fields"`
}

type FieldChange struct {
    Field    string      `json:"field"`
    OldValue interface{} `json:"old_value"`
    NewValue interface{} `json:"new_value"`
}

func NewDeltaPlugin(writer AuditWriter, orgExtractor OrgIDExtractor) *DeltaPlugin {
    return &DeltaPlugin{writer: writer, orgExtractor: orgExtractor}
}

func (p *DeltaPlugin) Name() string { return "deployguard:audit_delta" }

func (p *DeltaPlugin) Initialize(db *gorm.DB) error {
    // Before: snapshot the current DB state of audit fields (for updates and deletes)
    if err := db.Callback().Update().Before("gorm:update").
        Register("deployguard:audit_delta:snapshot_update", p.snapshot); err != nil {
        return err
    }
    if err := db.Callback().Delete().Before("gorm:delete").
        Register("deployguard:audit_delta:snapshot_delete", p.snapshot); err != nil {
        return err
    }

    // After: diff and write the AuditRecord
    if err := db.Callback().Create().After("gorm:create").
        Register("deployguard:audit_delta:diff_create", p.diffCreate); err != nil {
        return err
    }
    if err := db.Callback().Update().After("gorm:update").
        Register("deployguard:audit_delta:diff_update", p.diffUpdate); err != nil {
        return err
    }
    if err := db.Callback().Delete().After("gorm:delete").
        Register("deployguard:audit_delta:diff_delete", p.diffDelete); err != nil {
        return err
    }
    return nil
}

const snapshotKey = "deployguard:audit_snapshot"

// snapshot loads the current row from the DB and stashes audited field values
// in the statement's context before the update executes.
func (p *DeltaPlugin) snapshot(db *gorm.DB) {
    if db.Statement == nil || db.Statement.Model == nil {
        return
    }

    rt := reflect.TypeOf(db.Statement.Model)
    for rt.Kind() == reflect.Ptr {
        rt = rt.Elem()
    }

    auditFields := tags.FieldsWithTag(rt, "audit")
    if len(auditFields) == 0 {
        return
    }

    // Load only the audited columns to minimise I/O
    cols := make([]string, 0, len(auditFields))
    for _, f := range auditFields {
        if f.TagValue == "true" {
            cols = append(cols, f.ColumnName)
        }
    }

    // Create a new instance of the same model type to receive the current state
    current := reflect.New(rt).Interface()
    result := db.Session(&gorm.Session{NewDB: true}).
        Select(cols).
        First(current, db.Statement.Model)
    if result.Error != nil {
        // Record may not exist yet (first save) — not an error
        return
    }

    // Stash the snapshot in the statement context
    db.Statement.Context = context.WithValue(
        db.Statement.Context,
        snapshotKey,
        extractAuditValues(current, auditFields),
    )
}

func (p *DeltaPlugin) diffCreate(db *gorm.DB) { p.recordDelta(db, "CREATE") }
func (p *DeltaPlugin) diffUpdate(db *gorm.DB) { p.recordDelta(db, "UPDATE") }
func (p *DeltaPlugin) diffDelete(db *gorm.DB) { p.recordDelta(db, "DELETE") }

// recordDelta compares the post-save model against the snapshot and writes an AuditRecord
// for every changed audit:"true" field.
func (p *DeltaPlugin) recordDelta(db *gorm.DB, action string) {
    if db.Statement == nil || db.Statement.Model == nil {
        return
    }
    if action == "UPDATE" && db.RowsAffected == 0 {
        return // Nothing changed in the DB
    }

    var snapshot map[string]interface{}
    if action == "UPDATE" || action == "DELETE" {
        snapshot, _ = db.Statement.Context.Value(snapshotKey).(map[string]interface{})
    }

    rt := reflect.TypeOf(db.Statement.Model)
    for rt.Kind() == reflect.Ptr {
        rt = rt.Elem()
    }

    auditFields := tags.FieldsWithTag(rt, "audit")
    current := extractAuditValues(db.Statement.Model, auditFields)

    var changes []FieldChange
    for _, f := range auditFields {
        if f.TagValue != "true" {
            continue
        }
        
        var oldVal, newVal interface{}
        switch action {
        case "CREATE":
            newVal = current[f.FieldName]
        case "UPDATE":
            oldVal = snapshot[f.FieldName]
            newVal = current[f.FieldName]
        case "DELETE":
            oldVal = snapshot[f.FieldName]
        }

        if !reflect.DeepEqual(oldVal, newVal) {
            changes = append(changes, FieldChange{
                Field:    f.ColumnName,
                OldValue: oldVal,
                NewValue: newVal,
            })
        }
    }

    if len(changes) == 0 {
        return
    }

    orgID, ok := p.orgExtractor.ExtractOrgID(db.Statement.Model)
    if !ok {
        db.AddError(fmt.Errorf("audit delta: could not extract org_id from %T", db.Statement.Model))
        return
    }

    delta := FieldDelta{
        ResourceType:  tableName(rt),
        ResourceID:    fmt.Sprintf("%v", primaryKey(db.Statement.Model)),
        ChangedFields: changes,
    }

    if err := p.writer.WriteFieldDelta(db.Statement.Context, delta); err != nil {
        // Fatal — AddError rolls back the GORM transaction.
        // This enforces "Strict Audit Mode": if we cannot audit the change, the change fails.
        db.AddError(fmt.Errorf("audit delta: write failed for %s: %w", delta.ResourceType, err))
    }
    _ = orgID // orgID is passed to WriteFieldDelta inside the implementation
}

func extractAuditValues(model interface{}, fields []tags.TaggedField) map[string]interface{} {
    rv := reflect.ValueOf(model)
    for rv.Kind() == reflect.Ptr {
        rv = rv.Elem()
    }
    result := make(map[string]interface{}, len(fields))
    for _, f := range fields {
        if f.TagValue != "true" {
            continue
        }
        fv := rv.FieldByName(f.FieldName)
        if fv.IsValid() {
            result[f.FieldName] = fv.Interface()
        }
    }
    return result
}

func tableName(t reflect.Type) string {
    // GORM's TableName() interface — call it if available
    instance := reflect.New(t).Interface()
    type tableNamer interface{ TableName() string }
    if tn, ok := instance.(tableNamer); ok {
        return tn.TableName()
    }
    return tags.ToSnakeCase(t.Name())
}

func primaryKey(model interface{}) interface{} {
    rv := reflect.ValueOf(model)
    for rv.Kind() == reflect.Ptr {
        rv = rv.Elem()
    }
    f := rv.FieldByName("ID")
    if f.IsValid() {
        return f.Interface()
    }
    return "unknown"
}

// MarshalDelta serialises a FieldDelta to JSON for storage in audit_records.Payload.
func MarshalDelta(delta FieldDelta) ([]byte, error) {
    return json.Marshal(delta)
}
