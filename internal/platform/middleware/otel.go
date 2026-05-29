package middleware

import (
	"net/http"
)

// OTel OpenTelemetry tracing middleware scaffold
func OTel() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Inject context with OpenTelemetry span
			// ctx, span := otel.Tracer("http").Start(r.Context(), r.URL.Path)
			// defer span.End()
			// next.ServeHTTP(w, r.WithContext(ctx))

			next.ServeHTTP(w, r)
		})
	}
}
