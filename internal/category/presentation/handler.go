package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/category/application"
	"github.com/trvux/elc-go/internal/category/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type CategoryHandler struct {
	repo domain.CategoryRepository
}

func NewCategoryHandler(repo domain.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{repo: repo}
}

func filterFromQuery(r *http.Request) (domain.CategoryFilter, error) {
	filter := domain.CategoryFilter{
		Search:         r.URL.Query().Get("search"),
		GroupID:        r.URL.Query().Get("group_id"),
		IncludeDeleted: r.URL.Query().Get("include_deleted") == "true",
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return filter, apperr.NewValidationError("invalid limit query param", nil)
		}
		filter.Limit = limit
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil {
			return filter, apperr.NewValidationError("invalid offset query param", nil)
		}
		filter.Offset = offset
	}
	return filter, nil
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := filterFromQuery(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	categories, err := application.GetCategories(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toCategoryResponseList(categories))
}

func (h *CategoryHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter, err := filterFromQuery(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	count, err := application.CountCategories(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *CategoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	c, err := application.GetCategoryByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if c == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("category"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toCategoryResponse(c))
}

func (h *CategoryHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	c, err := application.GetCategoryBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if c == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("category"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toCategoryResponse(c))
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateCategoryInput{
		Name:            req.Name,
		Slug:            req.Slug,
		GroupID:         req.GroupID,
		ImageURL:        req.ImageURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
	}

	c, err := application.CreateCategory(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toBareCategoryResponse(c))
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateCategoryInput{
		ID:              id,
		Name:            req.Name,
		Slug:            req.Slug,
		GroupID:         req.GroupID,
		ImageURL:        req.ImageURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
	}

	c, err := application.UpdateCategory(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBareCategoryResponse(c))
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteCategory(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CategoryHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreCategory(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
