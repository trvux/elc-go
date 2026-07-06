package presentation

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts the event module's routes — both public, consistent
// with every other GET in this codebase (no CORS anywhere, only ever called
// server-side from Next.js — see project notes). Create is a public write
// too: the beacon comes from anonymous site visitors, defended by the rate
// limiter inside the handler rather than auth.
func RegisterRoutes(r chi.Router, h *EventHandler) {
	r.Route("/events", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/top-viewed", h.TopViewed)
	})
}
