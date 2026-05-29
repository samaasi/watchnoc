package tenant

import (
    "context"
    "fmt"
    "net/http"
    "reflect"

    "gorm.io/gorm"

    "github.com/samaasi/watchnoc/internal/platform/tags"
)

// orgIDContextKey is the unexported context key for the org ID.
type orgIDContextKey struct{}

// WithOrgID stores the authenticated org ID in the context.
// Called by the auth middleware after JWT/session validation.
func WithOrgID(ctx context.Context, orgID uint64) context.Context {
    return context.WithValue(ctx, orgIDContextKey{}, orgID)
}

// OrgIDFromContext retrieves the org ID from the context.
// Returns 0 and false if no org ID is present.
func OrgIDFromContext(ctx context.Context) (uint64, bool) {
    id, ok := ctx.Value(orgIDContextKey{}).(uint64)
    return id, ok && id > 0
}

// Scope returns a GORM scope function that appends WHERE org_id = ? to the query.
// Must only be used with models that carry tenant:"org_id" — this is verified at
// runtime in development (panics) and logs a warning in production.
//
// Usage:
//
//	var events []deploy.DeployEvent
//	db.WithContext(ctx).
//	    Scopes(tenant.Scope(orgID)).
//	    Where("status = ?", "pending").
//	    Find(&events)
func Scope(orgID uint64) func(*gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        if db.Statement == nil {
            return db.Where("org_id = ?", orgID)
        }
        // Validate at runtime that the model has a tenant tag
        if db.Statement.Model != nil {
            rt := reflect.TypeOf(db.Statement.Model)
            for rt.Kind() == reflect.Ptr {
                rt = rt.Elem()
            }
            if rt.Kind() == reflect.Struct && !tags.HasTag(rt, "tenant") {
                // Non-tenanted model — bug in the call site
                db.AddError(fmt.Errorf(
                    "tenant scope applied to non-tenanted model %s — missing tenant:\"org_id\" tag",
                    rt.Name(),
                ))
                return db
            }
        }
        return db.Where("org_id = ?", orgID)
    }
}

// ScopeFromContext reads the org ID from ctx and applies Scope.
// Convenience wrapper for repository methods that receive a context.
//
// Returns an error scope (query will fail) if no org ID is present in ctx —
// this is always a caller bug in an authenticated handler.
func ScopeFromContext(ctx context.Context) func(*gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        orgID, ok := OrgIDFromContext(ctx)
        if !ok {
            db.AddError(fmt.Errorf(
                "tenant: no org_id in context — ensure auth middleware ran before this query",
            ))
            return db
        }
        return Scope(orgID)(db)
    }
}

// TenantMiddleware is an HTTP middleware that reads org_id from the authenticated
// JWT claims and injects it into the request context via WithOrgID.
// Must run after the authentication middleware.
func TenantMiddleware(claimsExtractor ClaimsExtractor) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims, ok := claimsExtractor.Claims(r)
            if !ok {
                // Auth middleware should have already rejected unauthenticated requests
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            orgID := claims.OrgID
            if orgID == 0 {
                http.Error(w, "missing org context", http.StatusUnauthorized)
                return
            }

            ctx := WithOrgID(r.Context(), orgID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// ClaimsExtractor is satisfied by the auth middleware's claims provider.
type ClaimsExtractor interface {
    Claims(r *http.Request) (OrgClaims, bool)
}

type OrgClaims struct {
    OrgID  uint64
    UserID uint64
    Role   string
}
