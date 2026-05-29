package retention_test

import (
    "reflect"
    "testing"

    "github.com/stretchr/testify/assert"

    "github.com/samaasi/watchnoc/internal/platform/retention"
)

type exemptModel struct {
    OrgID uint64 `hard_purge:"f"`
}

type purgableModel struct {
    OrgID uint64
}

func TestExemptionRegistry_IsExempt(t *testing.T) {
    registry := retention.NewExemptionRegistry(
        retention.ModelEntry{Table: "audit_records", Type: reflect.TypeOf(exemptModel{})},
        retention.ModelEntry{Table: "deploy_events", Type: reflect.TypeOf(purgableModel{})},
    )

    assert.True(t, registry.IsExempt("audit_records"))
    assert.False(t, registry.IsExempt("deploy_events"))
    assert.False(t, registry.IsExempt("unknown_table"))
}

func TestExemptionRegistry_Validate(t *testing.T) {
    registry := retention.NewExemptionRegistry(
        retention.ModelEntry{Table: "retention_ledger", Type: reflect.TypeOf(exemptModel{})},
    )

    err := registry.Validate("retention_ledger")
    assert.Error(t, err)

    err = registry.Validate("deploy_events")
    assert.NoError(t, err)
}
