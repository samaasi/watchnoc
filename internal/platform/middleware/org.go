package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	platformerrors "github.com/samaasi/watchnoc/internal/platform/errors"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

// OrgCtx middleware extracts the orgID from the URL and adds it to the context.
func OrgCtx(responder *response.ChiResponder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgIDStr := chi.URLParam(r, "orgID")
			if orgIDStr == "" {
				responder.Error(w, r, platformerrors.Validation("MISSING_ORG_ID", "missing orgID parameter"))
				return
			}
			orgID, err := strconv.ParseUint(orgIDStr, 10, 64)
			if err != nil {
				responder.Error(w, r, platformerrors.Validation("INVALID_ORG_ID", "invalid orgID parameter"))
				return
			}

			ctx := context.WithValue(r.Context(), OrgIDKey, orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetOrgID extracts the orgID from the context.
func GetOrgID(ctx context.Context) (uint64, bool) {
	orgID, ok := ctx.Value(OrgIDKey).(uint64)
	return orgID, ok
}

// GetUserID extracts the userID from the context.
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
