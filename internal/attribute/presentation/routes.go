package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

func RegisterRoutes(r chi.Router, h *AttributeDefinitionHandler, verifier httpserver.TokenVerifier) {
	r.Route("/attribute-definitions", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Post("/", h.Create)
			r.Put("/{id}", h.Update)
			r.Post("/{id}/restore", h.Restore)
			r.Post("/{id}/categories", h.AttachCategories)
			r.Delete("/{id}/categories/{categoryId}", h.DetachCategory)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/{id}", h.Delete)
		})
	})
}
