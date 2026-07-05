package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the contact module's routes onto the shared router
// built by cmd/server/main.go. Reads stay public (the public site needs
// them); writes require a valid session with content:write/content:delete.
func RegisterRoutes(r chi.Router, h *ContactHandler, verifier httpserver.TokenVerifier) {
	r.Route("/contacts", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Post("/", h.Create)
			r.Put("/{id}", h.Update)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/{id}", h.Delete)
		})
	})
}
