package middleware

import (
	"net/http"

	"github.com/samaasi/watchnoc/internal/platform/response"
)

// RateLimit basic token bucket rate limiter scaffold
func RateLimit(responder *response.ChiResponder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Implement redis-backed rate limiting per tenant/user
			// If rate limited:
			// responder.Error(w, r, errors.ErrRateLimited) // Note: ErrRateLimited must be added to definitions.go
			// return

			next.ServeHTTP(w, r)
		})
	}
}
