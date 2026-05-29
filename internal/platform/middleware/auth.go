package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/samaasi/watchnoc/internal/platform/errors"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

const UserIDKey contextKey = "user_id"
const OrgIDKey contextKey = "org_id"

// Auth validation middleware
func Auth(responder *response.ChiResponder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				responder.Error(w, r, errors.ErrUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				responder.Error(w, r, errors.ErrUnauthorized)
				return
			}

			claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{
				Token: token,
			})
			if err != nil {
				responder.Error(w, r, errors.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
