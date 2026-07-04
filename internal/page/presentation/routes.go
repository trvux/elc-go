package presentation

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *PageHandler) {
	r.Route("/pages", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/count", h.Count)
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Get("/slug/{slug}", h.GetBySlug)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/restore", h.Restore)
	})
}
