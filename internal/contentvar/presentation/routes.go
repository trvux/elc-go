package presentation

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *ContentVarHandler) {
	r.Route("/content-variables", func(r chi.Router) {
		// POST but read-only (resolves against the live catalog, writes
		// nothing) — called by the public site's own SSR render step, same
		// reasoning as /products/by-ids, so it stays open.
		r.Post("/resolve", h.ResolveVariables)
	})
}
