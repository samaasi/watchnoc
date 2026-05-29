package observability

import (
	"net/http"
	"time"
)

// InitMetrics initializes and registers basic Prometheus metrics
func InitMetrics() {
	// Scaffold for initializing metrics
	// e.g. prometheus.MustRegister(...)
}

// RecordRequestMetrics captures standard HTTP metrics (duration, status, etc.)
func RecordRequestMetrics(method, path string, status int, duration time.Duration) {
	// Scaffold for observing metrics
	// e.g. requestDuration.WithLabelValues(method, path, strconv.Itoa(status)).Observe(duration.Seconds())
}

// MetricsHandler returns an HTTP handler for exposing Prometheus metrics
func MetricsHandler() http.Handler {
	// return promhttp.Handler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("metrics stub"))
	})
}
