package retention

import (
	"fmt"
	"reflect"

	"github.com/samaasi/watchnoc/internal/platform/tags"
)

// DeletionColumn determines which column a given model uses for deletion
// based on its tombstone tag. It returns an empty string if the model
// uses a physical hard purge (i.e. no soft-delete column).
func DeletionColumn(modelType reflect.Type) (string, error) {
	fields := tags.FieldsWithTag(modelType, "tombstone")
	for _, f := range fields {
		switch f.TagValue {
		case "void":
			return f.ColumnName, nil // typically "voided_at"
		case "soft_delete":
			return "deleted_at", nil
		case "hard_purge":
			return "", nil // No deletion column, skip soft-delete phase
		}
	}
	return "", fmt.Errorf("model lacks a tombstone tag; retention worker cannot determine deletion column")
}
