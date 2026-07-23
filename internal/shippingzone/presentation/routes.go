package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

func RegisterRoutes(r chi.Router, h *ShippingZoneHandler, verifier httpserver.TokenVerifier) {
	r.Route("/shipping", func(r chi.Router) {
		r.Get("/provinces", h.ListProvinces)
		r.Get("/wards", h.ListWards)
		r.Get("/zones", h.List)
		r.Get("/zones/default", h.Default)
		r.Get("/zones/lookup", h.Lookup)
		r.Get("/zones/{id}", h.GetByID)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Post("/zones", h.Create)
			r.Put("/zones/{id}", h.Update)
			r.Post("/provinces", h.CreateProvince)
			r.Put("/provinces/{code}", h.UpdateProvince)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/zones/{id}", h.Delete)
			r.Delete("/provinces/{code}", h.DeleteProvince)
		})
	})
}
