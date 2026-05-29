package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Handler handles health check endpoints.
type Handler struct {
	DB  *gorm.DB
	RDB *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{DB: db, RDB: rdb}
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

	// Check Redis health
	if h.RDB != nil {
		if err := h.RDB.Ping(r.Context()).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Redis ping failed"))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
