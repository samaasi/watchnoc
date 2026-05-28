package middleware

import (
	"net/http"

	"github.com/samaasi/watchnoc/internal/platform/errors"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

func ErrorHandler(resp *response.ChiResponder) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The Recover middleware handles panics, which calls resp.Error
			next.ServeHTTP(w, r)
		})
	}
}

// NotFoundHandler is a custom 404 handler
func NotFoundHandler(resp *response.ChiResponder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp.Error(w, r, errors.NotFound("DG-NOTFOUND-001", "Route not found"))
	}
}

// MethodNotAllowedHandler is a custom 405 handler
func MethodNotAllowedHandler(resp *response.ChiResponder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp.Error(w, r, errors.New("DG-METHOD-001", "Method not allowed", "Method not allowed", errors.CategoryValidation))
	}
}
