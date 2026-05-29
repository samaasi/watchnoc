package audit

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

// Handler defines audit HTTP endpoints
type Handler struct {
	service   Service
	responder response.Responder
}

func NewHandler(service Service, responder response.Responder) *Handler {
	return &Handler{service: service, responder: responder}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/audit", h.handleListAuditRecords)
	r.Get("/audit/{id}", h.handleGetAuditRecord)
	r.Get("/audit/export", h.handleExportAuditTrail)
	r.Post("/audit/verify-chain", h.handleVerifyAuditChain)
}

func (h *Handler) handleListAuditRecords(w http.ResponseWriter, r *http.Request) {
	// TODO: Get orgID from context
	// TODO: Parse limit and offset
	h.responder.Success(w, r, map[string]interface{}{"audit_records": []interface{}{}})
}

func (h *Handler) handleGetAuditRecord(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	_, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.responder.Error(w, r, ErrAuditRecordNotFound)
		return
	}
	// TODO: Get orgID from context
	h.responder.Success(w, r, map[string]interface{}{"audit_record": nil})
}

func (h *Handler) handleExportAuditTrail(w http.ResponseWriter, r *http.Request) {
	// TODO: Get orgID from context
	// TODO: Parse from and to dates
	h.responder.Success(w, r, map[string]interface{}{"audit_trail": []interface{}{}})
}

func (h *Handler) handleVerifyAuditChain(w http.ResponseWriter, r *http.Request) {
	// TODO: Get orgID from context
	h.responder.Success(w, r, map[string]interface{}{"valid": true})
}
