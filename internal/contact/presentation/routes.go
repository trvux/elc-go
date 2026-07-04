package presentation

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts the contact module's routes onto the shared router
// built by cmd/server/main.go.
func RegisterRoutes(r chi.Router, h *ContactHandler) {
	r.Route("/contacts", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}
