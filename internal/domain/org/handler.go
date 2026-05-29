package org

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

// Handler defines org HTTP endpoints
type Handler struct {
	service   Service
	responder response.Responder
}

func NewHandler(service Service, responder response.Responder) *Handler {
	return &Handler{service: service, responder: responder}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/org", h.handleGetOrg)
	r.Patch("/org/settings", h.handleUpdateSettings)
	r.Get("/org/members", h.handleGetMembers)
	r.Post("/org/members/invite", h.handleInviteMember)
}

func (h *Handler) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	// TODO: Get org ID from context, fetch org, return
	h.responder.Success(w, r, nil)
}

func (h *Handler) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse request, update org settings
	h.responder.NoContent(w, r)
}

func (h *Handler) handleGetMembers(w http.ResponseWriter, r *http.Request) {
	// TODO: Get org ID, fetch members, return
	h.responder.Success(w, r, nil)
}

func (h *Handler) handleInviteMember(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse invite request, add member
	h.responder.Created(w, r, nil)
}
