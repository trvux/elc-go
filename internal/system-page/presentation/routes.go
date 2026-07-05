package presentation

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *SystemPageHandler) {
	r.Route("/system-pages", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/slug/{slug}", h.GetBySlug)
		r.Put("/{id}", h.Update)
	})
}
