package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// Handler handles health check endpoints.
type Handler struct {
	DB *gorm.DB
	// Redis client would go here too
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/health", h.Health)
	r.Get("/health/live", h.Live)
	r.Get("/health/ready", h.Ready)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	// Check Postgres health
	sqlDB, err := h.DB.DB()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("DB unavailable"))
		return
	}

	if err := sqlDB.PingContext(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("DB ping failed"))
		return
	}

	// TODO: Add Redis ping when Redis is set up

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
