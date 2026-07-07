package presentation

import (
	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// RegisterRoutes mounts the image upload endpoint used by the admin panel's
// forms (product/category/brand/... image fields). Auth-gated like every
// other write route — see docs/auth.md.
func RegisterRoutes(r chi.Router, h *UploadHandler, verifier httpserver.TokenVerifier) {
	r.Group(func(r chi.Router) {
		r.Use(httpserver.RequireAuth(verifier))
		r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
		r.Post("/uploads", h.Upload)
	})
}
