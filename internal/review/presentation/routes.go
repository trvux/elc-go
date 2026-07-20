package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts review's routes. The public /reviews/{entityType}/
// {entityID} routes are anonymous (site visitors submit/read reviews).
// /admin/reviews is staff-only — every review regardless of entity type,
// with reviewer_phone included — gated the same way auth's own
// /admin/invites and /admin/users are (see internal/auth/presentation/
// routes.go). Reuses authdomain.CanWriteContent rather than adding a
// dedicated "reviews:manage" permission, same reasoning as internal/
// inquiry's routes.go: access doesn't need to diverge from content-write
// yet.
func RegisterRoutes(r chi.Router, h *ReviewHandler, verifier httpserver.TokenVerifier) {
	r.Route("/reviews/{entityType}/{entityID}", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
	})

	r.Route("/admin/reviews", func(r chi.Router) {
		r.Use(httpserver.RequireAuth(verifier))
		r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
		r.Get("/", h.AdminList)
		r.Get("/count", h.AdminCount)
	})
}
