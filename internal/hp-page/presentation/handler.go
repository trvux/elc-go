package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/hp-page/application"
	"github.com/trvux/elc-go/internal/hp-page/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type HpPageHandler struct {
	repo domain.HpPageRepository
}

func NewHpPageHandler(repo domain.HpPageRepository) *HpPageHandler {
	return &HpPageHandler{repo: repo}
}

func (h *HpPageHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.HpPageFilter{
		Search:         r.URL.Query().Get("search"),
		IncludeDeleted: r.URL.Query().Get("include_deleted") == "true",
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			httpserver.WriteError(w, apperr.NewValidationError("invalid limit query param", nil))
			return
		}
		filter.Limit = limit
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil {
			httpserver.WriteError(w, apperr.NewValidationError("invalid offset query param", nil))
			return
		}
		filter.Offset = offset
	}

	pages, err := application.GetHpPages(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toHpPageResponseList(pages))
}

func (h *HpPageHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := application.GetHpPageByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if p == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("hp_page"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toHpPageResponse(p))
}

func (h *HpPageHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	p, err := application.GetHpPageBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if p == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("hp_page"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toHpPageResponse(p))
}

func (h *HpPageHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createHpPageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateHpPageInput{
		Name:            req.Name,
		Slug:            req.Slug,
		ImageURL:        req.ImageURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
		AttributeCode:   req.AttributeCode,
		AttributeValues: req.AttributeValues,
	}

	p, err := application.CreateHpPage(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toHpPageResponse(p))
}

func (h *HpPageHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateHpPageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateHpPageInput{
		ID:              id,
		Name:            req.Name,
		Slug:            req.Slug,
		ImageURL:        req.ImageURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
		AttributeCode:   req.AttributeCode,
		AttributeValues: req.AttributeValues,
	}

	p, err := application.UpdateHpPage(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toHpPageResponse(p))
}

func (h *HpPageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteHpPage(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HpPageHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreHpPage(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
