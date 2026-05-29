package approval

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

// Handler defines approval HTTP endpoints
type Handler struct {
	service   Service
	responder response.Responder
}

func NewHandler(service Service, responder response.Responder) *Handler {
	return &Handler{service: service, responder: responder}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/approvals", h.handleListApprovals)
	r.Get("/approvals/{id}", h.handleGetApproval)
	r.Post("/approvals/{id}/grant", h.handleGrantApproval)
	r.Post("/approvals/{id}/reject", h.handleRejectApproval)
}

func (h *Handler) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	// TODO: Get orgID from context
	h.responder.Success(w, r, map[string]interface{}{"approvals": []interface{}{}})
}

func (h *Handler) handleGetApproval(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	_, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.responder.Error(w, r, ErrApprovalInvalidStatus)
		return
	}
	// TODO: Get orgID from context
	h.responder.Success(w, r, map[string]interface{}{"approval": nil})
}

func (h *Handler) handleGrantApproval(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	h.responder.NoContent(w, r)
}

func (h *Handler) handleRejectApproval(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	h.responder.NoContent(w, r)
}
