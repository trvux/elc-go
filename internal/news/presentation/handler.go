package presentation

import (
	"encoding/json"
	"net/http"

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

func parseNewsFilter(r *http.Request) (domain.NewsFilter, error) {
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

	limit, offset, err := httpserver.ParsePagination(r)
	if err != nil {
		return filter, err
	}
	filter.Limit = limit
	filter.Offset = offset
	return filter, nil
}

func (h *NewsHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseNewsFilter(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	items, err := application.GetNews(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toNewsResponseList(items))
}

func (h *NewsHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter, err := parseNewsFilter(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

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
		Images:          toImageAssetDomainList(req.Images),
		Content:         req.Content,
		Excerpt:         req.Excerpt,
		CategoryID:      req.CategoryID,
		AuthorID:        req.AuthorID,
		IsPublished:     req.IsPublished,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		OrderIndex:      req.OrderIndex,
		TagIDs:          req.TagIDs,
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

	input := domain.UpdateNewsInput{
		ID:              id,
		Title:           req.Title,
		Slug:            req.Slug,
		Images:          toImageAssetDomainList(req.Images),
		Content:         req.Content,
		Excerpt:         req.Excerpt,
		CategoryID:      req.CategoryID,
		AuthorID:        req.AuthorID,
		IsPublished:     req.IsPublished,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		OrderIndex:      req.OrderIndex,
		TagIDs:          req.TagIDs,
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
