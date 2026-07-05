package presentation

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *ProjectHandler) {
	r.Route("/projects", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/count", h.Count)
		r.Get("/adjacent", h.GetAdjacent)
		r.Get("/slug/{slug}", h.GetBySlug)
		r.Get("/categories-by-project-type/{projectTypeId}", h.GetCategoriesByProjectType)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/restore", h.Restore)
		r.Patch("/{id}/order", h.UpdateOrder)
		r.Patch("/{id}/publish", h.TogglePublish)
		r.Patch("/{id}/featured", h.ToggleFeatured)
	})
}
