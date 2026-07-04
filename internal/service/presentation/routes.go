package presentation

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *ServiceHandler) {
	r.Route("/services", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/count", h.Count)
		r.Post("/", h.Create)
		r.Get("/slug/{slug}", h.GetBySlug)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/restore", h.Restore)
	})
}
