package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

func RegisterRoutes(r chi.Router, h *SystemPageHandler, verifier httpserver.TokenVerifier) {
	r.Route("/system-pages", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/slug/{slug}", h.GetBySlug)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Put("/{id}", h.Update)
		})
	})
}
