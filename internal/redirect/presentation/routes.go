package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the redirect admin API. There is no public route: the
// redirects are applied by nginx, which reads the table directly, never by Next.
// Gated like the other staff-only content routes (CanWriteContent/CanDeleteContent).
func RegisterRoutes(r chi.Router, h *Handler, verifier httpserver.TokenVerifier) {
	r.Route("/admin/redirects", func(r chi.Router) {
		r.Use(httpserver.RequireAuth(verifier))

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Get("/", h.List)
			r.Post("/", h.Create)
			r.Put("/{id}", h.Update)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/{id}", h.Delete)
		})
	})
}
