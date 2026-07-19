package presentation

import (
	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the recently-viewed module's routes. Every route is
// public and unauthenticated — same as wishlist, keyed entirely by the
// anonymous visitor_id cookie EnsureVisitorID attaches to the request.
func RegisterRoutes(r chi.Router, h *RecentlyViewedHandler, secureCookies bool) {
	r.Route("/recently-viewed", func(r chi.Router) {
		r.Use(httpserver.EnsureVisitorID(secureCookies))
		r.Get("/", h.List)
		r.Post("/", h.Record)
	})
}
