// internal/platform/tags/tags_test.go

package tags_test

import (
    "reflect"
    "testing"

    "github.com/stretchr/testify/assert"

    "github.com/samaasi/watchnoc/internal/platform/tags"
)

type testModel struct {
    ID       uint64
    Name     string `pii:"true" audit:"true"`
    Email    string `pii:"true"`
    Token    string `crypto:"true" mask:"partial"`
    Hash     string `hash:"sha256"`
    OrgID    uint64 `tenant:"org_id"`
    Internal string `evidence_export:"false"`
    Normal   string
}

func TestFieldsWithTag_PII(t *testing.T) {
    fields := tags.FieldsWithTag(reflect.TypeOf(testModel{}), "pii")
    assert.Len(t, fields, 2)
    names := []string{fields[0].FieldName, fields[1].FieldName}
    assert.ElementsMatch(t, []string{"Name", "Email"}, names)
}

func TestFieldsWithTag_Crypto(t *testing.T) {
    fields := tags.FieldsWithTag(reflect.TypeOf(testModel{}), "crypto")
    assert.Len(t, fields, 1)
    assert.Equal(t, "Token", fields[0].FieldName)
    assert.Equal(t, "true", fields[0].TagValue)
}

func TestFieldsWithTag_Mask(t *testing.T) {
    fields := tags.FieldsWithTag(reflect.TypeOf(testModel{}), "mask")
    assert.Len(t, fields, 1)
    assert.Equal(t, "partial", fields[0].TagValue)
}

func TestFieldsWithTag_Tenant(t *testing.T) {
    fields := tags.FieldsWithTag(reflect.TypeOf(testModel{}), "tenant")
    assert.Len(t, fields, 1)
    assert.Equal(t, "org_id", fields[0].TagValue)
}

func TestFieldsWithTag_EvidenceExport(t *testing.T) {
    fields := tags.FieldsWithTag(reflect.TypeOf(testModel{}), "evidence_export")
    assert.Len(t, fields, 1)
    assert.Equal(t, "false", fields[0].TagValue)
}

func TestFieldsWithTag_NoTag(t *testing.T) {
    fields := tags.FieldsWithTag(reflect.TypeOf(testModel{}), "pii")
    for _, f := range fields {
        assert.NotEqual(t, "Normal", f.FieldName, "Normal field should not be tagged")
    }
}

func TestFieldsWithTag_Cache(t *testing.T) {
    // Call twice — should return identical results from cache
    first := tags.FieldsWithTag(reflect.TypeOf(testModel{}), "pii")
    second := tags.FieldsWithTag(reflect.TypeOf(testModel{}), "pii")
    assert.Equal(t, first, second)
}
