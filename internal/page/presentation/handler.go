package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/page/application"
	"github.com/trvux/elc-go/internal/page/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type PageHandler struct {
	repo domain.PageRepository
}

func NewPageHandler(repo domain.PageRepository) *PageHandler {
	return &PageHandler{repo: repo}
}

func (h *PageHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.PageFilter{
		Search: r.URL.Query().Get("search"),
	}

	if isPubStr := r.URL.Query().Get("is_published"); isPubStr != "" {
		isPub, err := strconv.ParseBool(isPubStr)
		if err == nil {
			filter.IsPublished = &isPub
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err == nil {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err == nil {
			filter.Offset = offset
		}
	}

	if inclDelStr := r.URL.Query().Get("include_deleted"); inclDelStr != "" {
		inclDel, err := strconv.ParseBool(inclDelStr)
		if err == nil {
			filter.IncludeDeleted = inclDel
		}
	}

	pages, err := application.GetPages(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toPageDTOList(pages))
}

func (h *PageHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter := domain.PageFilter{
		Search: r.URL.Query().Get("search"),
	}

	if isPubStr := r.URL.Query().Get("is_published"); isPubStr != "" {
		isPub, err := strconv.ParseBool(isPubStr)
		if err == nil {
			filter.IsPublished = &isPub
		}
	}

	if inclDelStr := r.URL.Query().Get("include_deleted"); inclDelStr != "" {
		inclDel, err := strconv.ParseBool(inclDelStr)
		if err == nil {
			filter.IncludeDeleted = inclDel
		}
	}

	count, err := application.CountPages(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *PageHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := application.GetPageByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if p == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("page"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toPageDTO(p))
}

func (h *PageHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	p, err := application.GetPageBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if p == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("page"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toPageDTO(p))
}

func (h *PageHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input domain.CreatePageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	p, err := application.CreatePage(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toPageDTO(p))
}

func (h *PageHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input domain.UpdatePageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}
	input.ID = id

	p, err := application.UpdatePage(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toPageDTO(p))
}

func (h *PageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := application.DeletePage(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PageHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := application.RestorePage(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
