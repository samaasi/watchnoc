package org

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
)

// Org error codes
const (
	ErrCodeOrgNotFound            = "DG-ORG-4040"
	ErrCodeOrgMemberAlreadyExists = "DG-ORG-4090"
)

// Org errors
var (
	ErrOrgNotFound            = errors.NotFound(ErrCodeOrgNotFound, "Organization not found")
	ErrOrgMemberAlreadyExists = errors.Conflict(ErrCodeOrgMemberAlreadyExists, "Member already exists in organization")
)
