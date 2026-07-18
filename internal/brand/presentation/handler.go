package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/brand/application"
	"github.com/trvux/elc-go/internal/brand/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type BrandHandler struct {
	repo domain.BrandRepository
}

func NewBrandHandler(repo domain.BrandRepository) *BrandHandler {
	return &BrandHandler{repo: repo}
}

func (h *BrandHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.BrandFilter{
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

	brands, err := application.GetBrands(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBrandResponseList(brands))
}

func (h *BrandHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	b, err := application.GetBrandByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if b == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("brand"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBrandResponse(b))
}

func (h *BrandHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	b, err := application.GetBrandBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if b == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("brand"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBrandResponse(b))
}

func (h *BrandHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBrandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateBrandInput{
		Name:            req.Name,
		Slug:            req.Slug,
		LogoURL:         req.LogoURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
		WarrantyPolicy:  req.WarrantyPolicy,
	}

	b, err := application.CreateBrand(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toBrandResponse(b))
}

func (h *BrandHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateBrandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateBrandInput{
		ID:              id,
		Name:            req.Name,
		Slug:            req.Slug,
		LogoURL:         req.LogoURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
		WarrantyPolicy:  req.WarrantyPolicy,
	}

	b, err := application.UpdateBrand(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBrandResponse(b))
}

func (h *BrandHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteBrand(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BrandHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreBrand(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
