package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

func RegisterRoutes(r chi.Router, h *ProductHandler, verifier httpserver.TokenVerifier) {
	r.Route("/products", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/count", h.Count)
		// POST but read-only (batch fetch by a body-carried ID list) — the
		// public site uses this too (cart/wishlist), so it stays open.
		r.Post("/by-ids", h.GetByIDsBatch)
		r.Get("/slug/{slug}", h.GetBySlug)
		r.Get("/{id}", h.GetByID)
		r.Get("/{id}/adjacent", h.GetAdjacent)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Post("/", h.Create)
			r.Put("/{id}", h.Update)
			r.Post("/{id}/restore", h.Restore)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/{id}", h.Delete)
		})
	})
}
