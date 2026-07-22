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
		r.Get("/compare", h.Compare)
		// POST but read-only (batch fetch by a body-carried ID list) — the
		// public site uses this too (cart/wishlist), so it stays open.
		r.Post("/by-ids", h.GetByIDsBatch)
		// Public, read-only, rate-limited in the handler itself (see
		// ProductHandler.chatSearchLimiter) since it's the one product
		// read path that runs a query per request rather than being
		// cacheable navigation.
		r.Post("/chat-search", h.ChatSearch)
		r.Get("/slug/{slug}", h.GetBySlug)
		r.Get("/{id}", h.GetByID)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Post("/", h.Create)
			r.Put("/{id}", h.Update)
			r.Post("/{id}/restore", h.Restore)
			r.Post("/{id}/submit", h.SubmitForReview)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanPublishContent))
			r.Post("/{id}/approve", h.Approve)
			r.Post("/{id}/reject", h.Reject)
			r.Post("/{id}/archive", h.Archive)
			r.Post("/{id}/unarchive", h.Unarchive)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/{id}", h.Delete)
		})
	})
}
