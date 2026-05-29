// internal/platform/tags/tags.go

package tags

import (
	"reflect"
	"strings"
	"sync"
)

// TaggedField describes a single struct field that carries a domain tag.
type TaggedField struct {
	// FieldName is the Go struct field name (e.g. "BotTokenEncrypted").
	FieldName string

	// ColumnName is the snake_case database column name derived from the
	// `gorm:"column:..."` tag, or snake_case(FieldName) if absent.
	ColumnName string

	// TagKey is the domain tag key (e.g. "crypto", "pii").
	TagKey string

	// TagValue is the tag value (e.g. "true", "argon2id", "partial", "org_id").
	TagValue string

	// Index is the field's position in the struct — used for reflection-based
	// get/set operations.
	Index int

	// Type is the reflect.Type of the field.
	Type reflect.Type
}

// cache avoids repeated reflection on the same type across requests.
var cache sync.Map // map[reflect.Type][]TaggedField

// FieldsWithTag returns all fields in a struct (or pointer-to-struct) that carry
// the given domain tag key. Results are cached after the first call per type.
//
// Usage:
//
//	fields := tags.FieldsWithTag(reflect.TypeOf(SlackInstallation{}), "crypto")
//	for _, f := range fields {
//	    // f.FieldName == "BotTokenEncrypted"
//	    // f.TagValue  == "true"
//	}
func FieldsWithTag(t reflect.Type, tagKey string) []TaggedField {
	// Dereference pointer types
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	cacheKey := cacheKeyFor(t, tagKey)
	if cached, ok := cache.Load(cacheKey); ok {
		return cached.([]TaggedField)
	}

	result := walkFields(t, tagKey)
	cache.Store(cacheKey, result)
	return result
}

// GetFieldValue returns the reflect.Value for a named field on a struct value.
// Panics if the field does not exist — callers should use FieldsWithTag first.
func GetFieldValue(v reflect.Value, fieldName string) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.FieldByName(fieldName)
}

// SetFieldValue sets a named field on a struct pointer to the given value.
// Returns false if the field is not settable.
func SetFieldValue(v reflect.Value, fieldName string, val interface{}) bool {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	f := v.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() {
		return false
	}
	f.Set(reflect.ValueOf(val))
	return true
}

// HasTag reports whether a struct type has at least one field with the given
// domain tag key. Used as a fast gate before more expensive reflection.
func HasTag(t reflect.Type, tagKey string) bool {
	return len(FieldsWithTag(t, tagKey)) > 0
}

// --- internal ---

type cacheKeyType struct {
	t      reflect.Type
	tagKey string
}

func cacheKeyFor(t reflect.Type, tagKey string) cacheKeyType {
	return cacheKeyType{t: t, tagKey: tagKey}
}

func walkFields(t reflect.Type, tagKey string) []TaggedField {
	var result []TaggedField
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Recurse into embedded structs (e.g. model.Base)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			embedded := walkFields(field.Type, tagKey)
			result = append(result, embedded...)
			continue
		}

		val, ok := field.Tag.Lookup(tagKey)
		if !ok {
			continue
		}

		result = append(result, TaggedField{
			FieldName:  field.Name,
			ColumnName: columnName(field),
			TagKey:     tagKey,
			TagValue:   val,
			Index:      i,
			Type:       field.Type,
		})
	}
	return result
}

// columnName derives the database column name from the gorm tag, falling
// back to a simple snake_case conversion of the field name.
func columnName(f reflect.StructField) string {
	gormTag := f.Tag.Get("gorm")
	for _, part := range strings.Split(gormTag, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}
	return ToSnakeCase(f.Name)
}

func ToSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}
