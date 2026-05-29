package ticket

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

type Handler struct {
	svc  Service
	resp *response.ChiResponder
}

func NewHandler(svc Service, resp *response.ChiResponder) *Handler {
	return &Handler{svc: svc, resp: resp}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/tickets", func(r chi.Router) {
		r.Get("/{key}", h.getTicket)
	})
}

func (h *Handler) getTicket(w http.ResponseWriter, r *http.Request) {
	// Scaffold logic
	w.WriteHeader(http.StatusNotImplemented)
}
