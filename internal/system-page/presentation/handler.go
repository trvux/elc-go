package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/system-page/application"
	"github.com/trvux/elc-go/internal/system-page/domain"
)

type SystemPageHandler struct {
	repo domain.SystemPageRepository
}

func NewSystemPageHandler(repo domain.SystemPageRepository) *SystemPageHandler {
	return &SystemPageHandler{repo: repo}
}

func (h *SystemPageHandler) List(w http.ResponseWriter, r *http.Request) {
	pages, err := application.GetSystemPages(r.Context(), h.repo)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toSystemPageDTOList(pages))
}

func (h *SystemPageHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	p, err := application.GetSystemPageBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if p == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("system page"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toSystemPageDTO(p))
}

func (h *SystemPageHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateSystemPageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateSystemPageInput{
		ID:              id,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
	}

	p, err := application.UpdateSystemPage(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toSystemPageDTO(p))
}
