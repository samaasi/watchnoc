package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

// Handler defines auth HTTP endpoints
type Handler struct {
	service   Service
	responder response.Responder
}

func NewHandler(service Service, responder response.Responder) *Handler {
	return &Handler{service: service, responder: responder}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/auth/logout", h.handleLogout)
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement logout (invalidate session, etc.)
	h.responder.NoContent(w, r)
}
