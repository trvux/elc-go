package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the inquiry module's routes. Create is public (the
// public site's lead-capture form submits anonymously) — everything else is
// staff-only. Reuses authdomain.CanWriteContent rather than adding a
// dedicated "leads:manage" permission: both currently grant the exact same
// {admin, user} set a new permission would, so a new constant would be
// unwarranted until lead-management access actually needs to diverge from
// content-write access.
func RegisterRoutes(r chi.Router, h *InquiryHandler, zaloWebhook *ZaloWebhookHandler, verifier httpserver.TokenVerifier) {
	r.Route("/inquiries", func(r chi.Router) {
		r.Post("/", h.Create)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Get("/", h.List)
			r.Get("/count", h.Count)
			r.Get("/{id}", h.GetByID)
			r.Patch("/{id}/status", h.UpdateStatus)
		})

		// Always mounted regardless of Zalo OA configuration state — see
		// ZaloWebhookHandler.Receive's doc comment.
		r.Post("/zalo/webhook", zaloWebhook.Receive)
	})
}
