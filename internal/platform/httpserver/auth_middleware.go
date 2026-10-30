package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

type ctxKey string

const (
	ctxKeyUserID ctxKey = "auth_user_id"
	ctxKeyRole   ctxKey = "auth_role"
)

// Claims is the minimal identity extracted from a verified access token.
type Claims struct {
	UserID string
	Role   string
}

// TokenVerifier is satisfied by the auth module's JWT issuer. It is declared
// here — not imported from internal/auth — so this cross-cutting platform
// package never depends on a specific business module; the composition root
// (cmd/server/main.go) injects the concrete implementation.
type TokenVerifier interface {
	Verify(tokenString string) (Claims, error)
}

// RequireAuth extracts the bearer access token, verifies it, and attaches the
// resulting claims to the request context. Responds 401 on any failure.
func RequireAuth(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, hasBearer := strings.CutPrefix(header, "Bearer ")
			if !hasBearer || token == "" {
				WriteError(w, apperr.NewUnauthorizedError("missing bearer token"))
				return
			}

			claims, err := verifier.Verify(token)
			if err != nil {
				WriteError(w, apperr.NewUnauthorizedError("invalid or expired token"))
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxKeyRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission must be mounted after RequireAuth. check is normally a
// closure over a business module's own permission map, e.g.:
//
//	httpserver.RequirePermission(func(role string) bool {
//	    return authdomain.Role(role).HasPermission(authdomain.PermissionContentWrite)
//	})
//
// This package takes a predicate instead of importing internal/auth/domain
// directly, for the same reason RequireAuth takes a TokenVerifier interface
// — this cross-cutting platform package must never depend on a specific
// business module.
func RequirePermission(check func(role string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := RoleFromContext(r.Context())
			if !check(role) {
				WriteError(w, apperr.NewForbiddenError("insufficient permission"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(string)
	return v, ok
}

func RoleFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyRole).(string)
	return v, ok
}
