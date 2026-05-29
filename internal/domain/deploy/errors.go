package deploy

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

// Deploy-related error codes
const (
	ErrCodeDeployNotFound      = "DG-DEPLOY-4040"
	ErrCodeDeployAlreadyExists = "DG-DEPLOY-4090"
	ErrCodeDeployInvalidStatus = "DG-DEPLOY-4000"
)

// Deploy errors
var (
	ErrDeployNotFound      = errors.NotFound(ErrCodeDeployNotFound, "Deployment not found")
	ErrDeployAlreadyExists = errors.Conflict(ErrCodeDeployAlreadyExists, "Deployment already exists")
	ErrDeployInvalidStatus = errors.Validation(ErrCodeDeployInvalidStatus, "Invalid deployment status")
)
