package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/tag/application"
	"github.com/trvux/elc-go/internal/tag/domain"
)

type TagHandler struct {
	repo domain.TagRepository
}

func NewTagHandler(repo domain.TagRepository) *TagHandler {
	return &TagHandler{repo: repo}
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.TagFilter{
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

	tags, err := application.GetTags(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toTagResponseList(tags))
}

func (h *TagHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	t, err := application.GetTagByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if t == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("tag"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toTagResponse(t))
}

func (h *TagHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	t, err := application.GetTagBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if t == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("tag"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toTagResponse(t))
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateTagInput{Name: req.Name, Slug: req.Slug}

	t, err := application.CreateTag(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toTagResponse(t))
}

func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateTagInput{ID: id, Name: req.Name, Slug: req.Slug}

	t, err := application.UpdateTag(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toTagResponse(t))
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteTag(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TagHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreTag(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
