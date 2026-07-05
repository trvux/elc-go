package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

func RegisterRoutes(r chi.Router, h *SettingsHandler, verifier httpserver.TokenVerifier) {
	r.Route("/settings", func(r chi.Router) {
		r.Get("/", h.List)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanManageSettings))
			r.Put("/", h.Update)
		})
	})
}
