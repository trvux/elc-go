package presentation

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the read-only slug lookup used by elc-tem to resolve
// which entity a public URL slug currently points to. No auth: this backs
// public page rendering, same as GetBySlug on the entity modules themselves.
func RegisterRoutes(r chi.Router, h *SlugRegistryHandler) {
	r.Get("/slug-registry/{slug}", h.GetBySlug)
}
