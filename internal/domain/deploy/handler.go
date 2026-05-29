package deploy

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/samaasi/watchnoc/internal/platform/errors"
	"github.com/samaasi/watchnoc/internal/platform/middleware"
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
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		h.responder.Error(w, r, errors.ErrUnauthorized)
		return
	}

	deploys, err := h.service.ListByOrg(r.Context(), orgID, 50, 0)
	if err != nil {
		h.responder.Error(w, r, errors.ErrInternalServer)
		return
	}

	h.responder.Success(w, r, map[string]interface{}{"data": deploys})
}

func (h *Handler) handleGetDeploy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		h.responder.Error(w, r, errors.ErrUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	deployID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.responder.Error(w, r, errors.Validation("INVALID_ID", "invalid deploy ID"))
		return
	}

	deploy, err := h.service.GetByID(r.Context(), orgID, deployID)
	if err != nil {
		if appErr, ok := err.(errors.AppError); ok && appErr.Code == ErrDeployNotFound.Code {
			h.responder.Error(w, r, errors.ErrNotFound)
			return
		}
		h.responder.Error(w, r, errors.ErrInternalServer)
		return
	}

	h.responder.Success(w, r, deploy)
}
