package response

import (
	"github.com/samaasi/watchnoc/internal/platform/errors"
	"encoding/json"
	"net/http"
	"time"
)

type Responder interface {
	Success(w http.ResponseWriter, r *http.Request, data interface{})
	Created(w http.ResponseWriter, r *http.Request, data interface{})
	NoContent(w http.ResponseWriter, r *http.Request)
	Error(w http.ResponseWriter, r *http.Request, err errors.AppError)
}

type ChiResponder struct{}

func NewChiResponder() *ChiResponder {
	return &ChiResponder{}
}

func (r *ChiResponder) Success(w http.ResponseWriter, req *http.Request, data interface{}) {
	respond(w, req, http.StatusOK, data, nil)
}

func (r *ChiResponder) Created(w http.ResponseWriter, req *http.Request, data interface{}) {
	respond(w, req, http.StatusCreated, data, nil)
}

func (r *ChiResponder) NoContent(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (r *ChiResponder) Error(w http.ResponseWriter, req *http.Request, err errors.AppError) {
	respond(w, req, err.HTTPStatus, nil, &err)
}

func respond(w http.ResponseWriter, req *http.Request, status int, data interface{}, err *errors.AppError) {
	meta := ResponseMeta{
		RequestID: getRequestID(req),
		TraceID:   getTraceID(req),
		Timestamp:  time.Now(),
		APIVersion: "v1",
	}

	var envelope interface{}
	if err != nil {
		envelope = Envelope[interface{}]{
			Success: false,
			Error: &ErrorDetail{
				Code:           err.Code,
				Message:        err.Message,
				UserMessage:    err.UserMessage,
				HTTPStatus:     err.HTTPStatus,
				ValidationErrors: err.ValidationErrors,
			},
			Meta: meta,
		}
	} else {
		envelope = Envelope[interface{}]{
			Success: true,
			Data:    data,
			Meta:    meta,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope)
}

func getRequestID(req *http.Request) string {
	if id := req.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return ""
}

func getTraceID(req *http.Request) string {
	if id := req.Header.Get("X-Trace-ID"); id != "" {
		return id
	}
	return ""
}
