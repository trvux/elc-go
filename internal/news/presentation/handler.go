package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/news/application"
	"github.com/trvux/elc-go/internal/news/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type NewsHandler struct {
	repo domain.NewsRepository
}

func NewNewsHandler(repo domain.NewsRepository) *NewsHandler {
	return &NewsHandler{repo: repo}
}

func parseNewsFilter(r *http.Request) domain.NewsFilter {
	q := r.URL.Query()
	filter := domain.NewsFilter{
		Search:         q.Get("search"),
		IncludeDeleted: q.Get("include_deleted") == "true",
		SortBy:         q.Get("sort_by"),
		SortOrder:      q.Get("sort_order"),
	}
	if v := q.Get("category_id"); v != "" {
		filter.CategoryID = &v
	}
	if v := q.Get("exclude_id"); v != "" {
		filter.ExcludeID = &v
	}
	if v := q.Get("is_published"); v != "" {
		b := v == "true"
		filter.IsPublished = &b
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}
	return filter
}

func (h *NewsHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := parseNewsFilter(r)

	items, err := application.GetNews(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toNewsResponseList(items))
}

func (h *NewsHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter := parseNewsFilter(r)

	count, err := application.CountNews(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, countResponse{Count: count})
}

func (h *NewsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	n, err := application.GetNewsByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if n == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("news"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toNewsResponse(n))
}

func (h *NewsHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	n, err := application.GetNewsBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if n == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("news"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toNewsResponse(n))
}

func (h *NewsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createNewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateNewsInput{
		Title:           req.Title,
		Slug:            req.Slug,
		Image:           req.Image,
		Content:         req.Content,
		CategoryID:      req.CategoryID,
		IsPublished:     req.IsPublished,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		Seo:             toSeoDomain(req.Seo),
		OrderIndex:      req.OrderIndex,
	}

	n, err := application.CreateNews(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toNewsResponse(n))
}

func (h *NewsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateNewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	var seo *domain.Seo
	if req.Seo != nil {
		s := toSeoDomain(*req.Seo)
		seo = &s
	}

	input := domain.UpdateNewsInput{
		ID:              id,
		Title:           req.Title,
		Slug:            req.Slug,
		Image:           req.Image,
		Content:         req.Content,
		CategoryID:      req.CategoryID,
		IsPublished:     req.IsPublished,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		Seo:             seo,
		OrderIndex:      req.OrderIndex,
	}

	n, err := application.UpdateNews(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toNewsResponse(n))
}

func (h *NewsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteNews(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *NewsHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreNews(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
