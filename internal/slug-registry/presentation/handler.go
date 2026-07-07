package presentation

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/slug-registry/application"
	"github.com/trvux/elc-go/internal/slug-registry/domain"
)

type SlugRegistryHandler struct {
	repo domain.SlugRegistryRepository
}

func NewSlugRegistryHandler(repo domain.SlugRegistryRepository) *SlugRegistryHandler {
	return &SlugRegistryHandler{repo: repo}
}

func (h *SlugRegistryHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	entry, err := application.GetSlugRegistryEntry(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if entry == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("slug registry entry"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toSlugRegistryEntryDTO(entry))
}
