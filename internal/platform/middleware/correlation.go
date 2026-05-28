package middleware

import (
	"context"
	"net/http"
)

type correlationKey string

const CorrelationIDKey correlationKey = "correlation_id"

func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			// TODO: generate a correlation ID if not present
		}
		ctx := context.WithValue(r.Context(), CorrelationIDKey, correlationID)
		w.Header().Set("X-Correlation-ID", correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
