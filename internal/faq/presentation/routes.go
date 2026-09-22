package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts faq's routes. The public /faqs/{ownerType}/
// {ownerID} read is anonymous (feeds the visible page + FAQPage JSON-LD).
// Everything else is staff-only, gated the same way internal/service-
// group's write routes are — reuses CanWriteContent/CanDeleteContent
// rather than a dedicated "faqs:manage" permission, same reasoning as
// internal/inquiry and internal/review's routes.go.
func RegisterRoutes(r chi.Router, h *FAQHandler, verifier httpserver.TokenVerifier) {
	r.Get("/faqs/{ownerType}/{ownerID}", h.List)

	r.Route("/admin/faqs", func(r chi.Router) {
		r.Use(httpserver.RequireAuth(verifier))

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Get("/{ownerType}/{ownerID}", h.AdminListByOwner)
			r.Post("/", h.Create)
			r.Put("/{id}", h.Update)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/{id}", h.Delete)
		})
	})
}
