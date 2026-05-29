package middleware

import (
	"net/http"

	"github.com/samaasi/watchnoc/internal/platform/response"
)

// RequireRole enforces role-based access control on routes
func RequireRole(role string, responder *response.ChiResponder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract user ID from context
			// userID, ok := r.Context().Value(UserIDKey).(string)
			
			// TODO: Look up user in DB, verify role membership
			// If not allowed:
			// responder.Error(w, r, errors.ErrForbidden)
			// return

			next.ServeHTTP(w, r)
		})
	}
}
