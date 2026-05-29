package audit

import (
	stdErrors "errors"

	"github.com/samaasi/watchnoc/internal/platform/errors"
)

// Audit-related error codes
const (
	ErrCodeAuditRecordNotFound = "DG-AUDIT-4040"
	ErrCodeAuditChainBroken    = "DG-AUDIT-5000"
)

// Audit errors
var (
	ErrAuditRecordNotFound = errors.NotFound(ErrCodeAuditRecordNotFound, "Audit record not found")
	ErrAuditChainBroken    = errors.Internal(ErrCodeAuditChainBroken, "Audit chain integrity compromised", stdErrors.New("audit chain broken"))
)
