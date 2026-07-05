package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

func RegisterRoutes(r chi.Router, h *ProjectTypeHandler, verifier httpserver.TokenVerifier) {
	r.Route("/project-types", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/count", h.Count)
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
