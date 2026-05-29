package approval

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

// Approval-related error codes
const (
	ErrCodeApprovalNotFound      = "DG-APPROVAL-4040"
	ErrCodeApprovalAlreadyExists = "DG-APPROVAL-4090"
	ErrCodeApprovalInvalidStatus = "DG-APPROVAL-4000"
)

// Approval errors
var (
	ErrApprovalNotFound      = errors.NotFound(ErrCodeApprovalNotFound, "Approval not found")
	ErrApprovalAlreadyExists = errors.Conflict(ErrCodeApprovalAlreadyExists, "Approval already exists")
	ErrApprovalInvalidStatus = errors.Validation(ErrCodeApprovalInvalidStatus, "Invalid approval status")
)
