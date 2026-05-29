package pii_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/samaasi/watchnoc/internal/platform/pii"
)

type piiModel struct {
	Email string `pii:"true"`
	Name  string `pii:"true"`
	Plan  string // not PII
}

func TestRedactStruct(t *testing.T) {
	m := piiModel{Email: "alice@acme.com", Name: "Alice Chen", Plan: "enterprise"}
	// Redacting is internal — test via the log handler or nullifier path
	// using FieldsWithTag
	fields := pii.CollectPIIColumns(reflect.TypeOf(m))
	assert.ElementsMatch(t, []string{"email", "name"}, fields)
	assert.NotContains(t, fields, "plan")
}

func TestNullificationDays_Constant(t *testing.T) {
	assert.Equal(t, 90, pii.PIINullificationDays,
		"PIINullificationDays must remain 90 — changing it affects GDPR compliance")
}
