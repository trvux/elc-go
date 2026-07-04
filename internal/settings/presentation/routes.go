package presentation

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *SettingsHandler) {
	r.Route("/settings", func(r chi.Router) {
		r.Get("/", h.List)
		r.Put("/", h.Update)
	})
}
