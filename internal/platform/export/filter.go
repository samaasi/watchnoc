// internal/platform/export/filter.go

package export

import (
    "reflect"
    "strings"

    "github.com/samaasi/watchnoc/internal/platform/tags"
)

// FilterForExport returns a shallow copy of the value with all
// evidence_export:"false" fields zeroed out.
//
// Works on both struct values and slices of structs. The original value
// is not modified — a new value is returned.
//
// Usage (in evidence PDF generator):
//
//	filtered := export.FilterForExport(deployEvent).(deploy.DeployEvent)
func FilterForExport(v interface{}) interface{} {
    if v == nil {
        return nil
    }
    rv := reflect.ValueOf(v)
    return filterValue(rv).Interface()
}

// FilterSliceForExport filters a slice of structs, applying FilterForExport to each element.
// Returns the filtered slice as an []interface{} — callers re-type as needed.
func FilterSliceForExport(slice interface{}) interface{} {
    rv := reflect.ValueOf(slice)
    if rv.Kind() != reflect.Slice {
        return FilterForExport(slice)
    }

    result := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
    for i := 0; i < rv.Len(); i++ {
        filtered := filterValue(rv.Index(i))
        result.Index(i).Set(filtered)
    }
    return result.Interface()
}

func filterValue(rv reflect.Value) reflect.Value {
    for rv.Kind() == reflect.Ptr {
        rv = rv.Elem()
    }
    if rv.Kind() != reflect.Struct {
        return rv
    }

    rt := rv.Type()
    excludedFields := tags.FieldsWithTag(rt, "evidence_export")

    // Only exclude fields explicitly tagged false
    exclusionSet := make(map[string]struct{})
    for _, f := range excludedFields {
        if strings.ToLower(f.TagValue) == "false" {
            exclusionSet[f.FieldName] = struct{}{}
        }
    }

    if len(exclusionSet) == 0 {
        return rv // Nothing to filter
    }

    // Create a settable copy
    copy := reflect.New(rt).Elem()
    copy.Set(rv)

    for fieldName := range exclusionSet {
        fv := copy.FieldByName(fieldName)
        if fv.IsValid() && fv.CanSet() {
            fv.Set(reflect.Zero(fv.Type()))
        }
    }

    return copy
}

// ExcludedColumns returns the list of DB column names that must be omitted from
// a SELECT query destined for an evidence export. Used to build the GORM
// .Omit(...) call in the evidence repository.
//
// Example:
//
//	omit := export.ExcludedColumns(reflect.TypeOf(deploy.DeployEvent{}))
//	db.Omit(omit...).Find(&events)
func ExcludedColumns(t reflect.Type) []string {
    fields := tags.FieldsWithTag(t, "evidence_export")
    var cols []string
    for _, f := range fields {
        if strings.ToLower(f.TagValue) == "false" {
            cols = append(cols, f.ColumnName)
        }
    }
    return cols
}
