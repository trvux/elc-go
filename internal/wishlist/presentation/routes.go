package presentation

import (
	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the wishlist module's routes. Every route here is
// public and unauthenticated — wishlist has no concept of an admin-panel
// account, it's keyed entirely by the anonymous visitor_id cookie
// EnsureVisitorID attaches to the request.
func RegisterRoutes(r chi.Router, h *WishlistHandler, secureCookies bool) {
	r.Route("/wishlist", func(r chi.Router) {
		r.Use(httpserver.EnsureVisitorID(secureCookies))
		r.Get("/", h.List)
		r.Post("/", h.Add)
		r.Delete("/{productId}", h.Remove)
	})
}
