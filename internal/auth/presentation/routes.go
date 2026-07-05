package presentation

import (
	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the auth module's routes onto the shared router.
// verifier is the same JWTTokenIssuer instance used to issue tokens — the
// composition root (cmd/server/main.go) is the only place that constructs it
// and wires it both here and into AuthHandler.
func RegisterRoutes(r chi.Router, h *AuthHandler, verifier httpserver.TokenVerifier) {
	r.Route("/auth", func(r chi.Router) {
		// Public — reachable by anyone, including a stranger with no
		// account. Each of these either does nothing without a valid
		// possession-proof (invite/reset token, correct password) or
		// deliberately reveals nothing about account existence.
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/refresh", h.Refresh)
		r.Post("/forgot-password", h.ForgotPassword)
		r.Post("/reset-password", h.ResetPassword)
		r.Post("/accept-invite", h.AcceptInvite)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Get("/me", h.Me)
		})
	})

	r.Route("/admin/invites", func(r chi.Router) {
		r.Use(httpserver.RequireAuth(verifier))
		r.Use(httpserver.RequirePermission(domain.CanManageAccounts))
		r.Post("/", h.CreateInvite)
	})

	r.Route("/admin/users", func(r chi.Router) {
		r.Use(httpserver.RequireAuth(verifier))
		r.Use(httpserver.RequirePermission(domain.CanManageAccounts))
		r.Get("/", h.ListUsers)
		r.Patch("/{id}", h.UpdateUser)
	})
}
