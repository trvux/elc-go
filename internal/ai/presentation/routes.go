package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the AI chat module's routes.
//
// /ai/chat is public/anonymous — the site's chat widget calls it without
// logging in — keyed by the visitor_id cookie EnsureVisitorID attaches
// (same pattern as wishlist/recently-viewed), plus OptionalAuth so a
// logged-in visitor's user_id gets attached to their conversation without
// requiring login. The chatLimiter inside AIHandler is the abuse defense.
//
// /ai/providers and /ai/models are the admin CRUD surface — gated behind
// CanManageSettings, the same predicate internal/settings already uses for
// site-wide config, rather than a new narrower permission (mirrors that
// module's own reasoning: nothing today needs the two to diverge).
func RegisterRoutes(r chi.Router, chatHandler *AIHandler, providerHandler *ProviderHandler, modelHandler *ModelHandler, verifier httpserver.TokenVerifier, secureCookies bool) {
	r.Route("/ai", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(httpserver.EnsureVisitorID(secureCookies))
			r.Use(httpserver.OptionalAuth(verifier))
			r.Post("/chat", chatHandler.Chat)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanManageSettings))

			r.Route("/providers", func(r chi.Router) {
				r.Get("/", providerHandler.List)
				r.Post("/", providerHandler.Create)
				r.Put("/{id}", providerHandler.Update)
				r.Delete("/{id}", providerHandler.Delete)
			})
			r.Route("/models", func(r chi.Router) {
				r.Get("/", modelHandler.List)
				r.Post("/", modelHandler.Create)
				r.Put("/{id}", modelHandler.Update)
				r.Delete("/{id}", modelHandler.Delete)
			})
		})
	})
}
