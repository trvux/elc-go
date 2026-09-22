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
func RegisterRoutes(r chi.Router, h *InquiryHandler, verifier httpserver.TokenVerifier) {
	r.Route("/inquiries", func(r chi.Router) {
		r.Post("/", h.Create)
		// Public — lets an anonymous visitor attach photos to a lead before
		// the inquiry itself exists. See UploadAttachment's doc comment.
		r.Post("/uploads", h.UploadAttachment)
		// Public — a Zalo/Messenger/Hotline contact-link click, called via
		// navigator.sendBeacon. See CreateClick's doc comment.
		r.Post("/clicks", h.CreateClick)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Get("/", h.List)
			r.Get("/count", h.Count)
			r.Get("/{id}", h.GetByID)
			r.Patch("/{id}/status", h.UpdateStatus)
		})
	})
}
