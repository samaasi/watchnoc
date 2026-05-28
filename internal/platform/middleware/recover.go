package middleware

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
	"github.com/samaasi/watchnoc/internal/platform/response"
	"log"
	"net/http"
	"runtime/debug"
)

func Recover(resp *response.ChiResponder) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("panic recovered: %v\n%s", rec, debug.Stack())
					resp.Error(w, r, errors.Internal("DG-INTERNAL-001", "Internal server error", nil))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
