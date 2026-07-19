package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	visitorCookieName   = "visitor_id"
	visitorCookieMaxAge = 365 * 24 * time.Hour

	ctxKeyVisitorID ctxKey = "visitor_id"
)

// EnsureVisitorID gives every anonymous site visitor a stable, opaque,
// server-issued ID across requests — no login involved. Used by Wishlist and
// Recently-Viewed, the first two modules with no other identity to key
// public writes on (see internal/wishlist, internal/recently-viewed). If the
// request already carries a visitor_id cookie, it's reused as-is; otherwise a
// new one is minted and set on the response before the handler runs.
// secure must be true in production (HTTPS) — same convention as auth's
// refresh_token cookie, see internal/auth/presentation/handler.go.
func EnsureVisitorID(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := ""
			if cookie, err := r.Cookie(visitorCookieName); err == nil && cookie.Value != "" {
				id = cookie.Value
			} else {
				id = uuid.NewString()
				http.SetCookie(w, &http.Cookie{
					Name:     visitorCookieName,
					Value:    id,
					Path:     "/",
					HttpOnly: true,
					Secure:   secure,
					SameSite: http.SameSiteLaxMode,
					MaxAge:   int(visitorCookieMaxAge.Seconds()),
				})
			}

			ctx := context.WithValue(r.Context(), ctxKeyVisitorID, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// VisitorIDFromContext reads the ID EnsureVisitorID attached to the request
// context — only populated on routes that mounted EnsureVisitorID.
func VisitorIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyVisitorID).(string)
	return v, ok
}
