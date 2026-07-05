package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

func RegisterRoutes(r chi.Router, h *ProjectHandler, verifier httpserver.TokenVerifier) {
	r.Route("/projects", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/count", h.Count)
		r.Get("/adjacent", h.GetAdjacent)
		r.Get("/slug/{slug}", h.GetBySlug)
		r.Get("/categories-by-project-type/{projectTypeId}", h.GetCategoriesByProjectType)
		r.Get("/{id}", h.GetByID)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Post("/", h.Create)
			r.Put("/{id}", h.Update)
			r.Post("/{id}/restore", h.Restore)
			r.Patch("/{id}/order", h.UpdateOrder)
			r.Patch("/{id}/publish", h.TogglePublish)
			r.Patch("/{id}/featured", h.ToggleFeatured)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/{id}", h.Delete)
		})
	})
}
