package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the chat-log module's routes. Create is public and
// anonymous — every chat-finder message logs itself, no login involved,
// keyed by the same visitor_id cookie recently-viewed/wishlist already use
// (see internal/platform/httpserver's EnsureVisitorID). Everything else
// (viewing the logged data for later analysis) is staff-only, same split
// as inquiry's own routes.go. Reuses authdomain.CanWriteContent rather
// than adding a dedicated permission — same reasoning as inquiry's routes.
func RegisterRoutes(r chi.Router, h *ChatLogHandler, verifier httpserver.TokenVerifier, secureCookies bool) {
	r.Route("/chat-logs", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(httpserver.EnsureVisitorID(secureCookies))
			r.Post("/", h.Create)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Get("/", h.List)
			r.Get("/count", h.Count)
		})
	})
}
