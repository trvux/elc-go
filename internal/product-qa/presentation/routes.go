package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts product-qa's routes under its own /questions prefix
// (not nested under /products, which internal/product already Mounts on this
// same chi.Router — mounting two subrouters on overlapping top-level
// patterns isn't supported). Ask/ListForProduct are public (anonymous site
// visitors submit/read questions) — everything else is staff-only, same
// permission reuse rationale as internal/inquiry's RegisterRoutes.
func RegisterRoutes(r chi.Router, h *QuestionHandler, verifier httpserver.TokenVerifier) {
	r.Route("/questions", func(r chi.Router) {
		r.Route("/product/{productId}", func(r chi.Router) {
			r.Get("/", h.ListForProduct)
			r.Post("/", h.Ask)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Get("/", h.List)
			r.Get("/count", h.Count)
			r.Post("/{id}/answer", h.Answer)
			r.Post("/{id}/reject", h.Reject)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/{id}", h.Delete)
		})
	})
}
