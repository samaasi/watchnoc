package deploy

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

// Handler defines deploy HTTP endpoints
type Handler struct {
	service   Service
	responder response.Responder
}

func NewHandler(service Service, responder response.Responder) *Handler {
	return &Handler{service: service, responder: responder}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/deploys", h.handleListDeploys)
	r.Get("/deploys/{id}", h.handleGetDeploy)
}

func (h *Handler) handleListDeploys(w http.ResponseWriter, r *http.Request) {
	// TODO: Get orgID from context
	// TODO: Parse limit and offset
	h.responder.Success(w, r, map[string]interface{}{"deploys": []interface{}{}})
}

func (h *Handler) handleGetDeploy(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	_, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.responder.Error(w, r, ErrDeployInvalidStatus)
		return
	}
	// TODO: Get orgID from context and find deploy
	h.responder.Success(w, r, map[string]interface{}{"deploy": nil})
}
