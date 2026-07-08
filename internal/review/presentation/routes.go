package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the review module's routes. Create/ListPublic/
// Summary are public (anonymous site visitors submit and read reviews) —
// everything else (the moderation screen) is staff-only. Reuses
// authdomain.CanWriteContent rather than adding a dedicated
// "reviews:manage" permission, same reasoning as inquiry's routes.go.
func RegisterRoutes(r chi.Router, h *ReviewHandler, verifier httpserver.TokenVerifier) {
	r.Route("/reviews", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/public", h.ListPublic)
		r.Get("/summary", h.Summary)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Get("/", h.List)
			r.Get("/count", h.Count)
			r.Get("/{id}", h.GetByID)
			r.Patch("/{id}/published", h.UpdatePublished)
			r.Delete("/{id}", h.Delete)
		})
	})
}
