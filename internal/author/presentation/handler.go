package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/author/application"
	"github.com/trvux/elc-go/internal/author/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type AuthorHandler struct {
	repo domain.AuthorRepository
}

func NewAuthorHandler(repo domain.AuthorRepository) *AuthorHandler {
	return &AuthorHandler{repo: repo}
}

func (h *AuthorHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.AuthorFilter{
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

	authors, err := application.GetAuthors(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toAuthorResponseList(authors))
}

func (h *AuthorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	a, err := application.GetAuthorByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if a == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("author"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toAuthorResponse(a))
}

func (h *AuthorHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	a, err := application.GetAuthorBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if a == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("author"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toAuthorResponse(a))
}

func (h *AuthorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createAuthorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateAuthorInput{
		Name:      req.Name,
		Slug:      req.Slug,
		AvatarURL: req.AvatarURL,
		Bio:       req.Bio,
	}

	a, err := application.CreateAuthor(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toAuthorResponse(a))
}

func (h *AuthorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateAuthorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateAuthorInput{
		ID:        id,
		Name:      req.Name,
		Slug:      req.Slug,
		AvatarURL: req.AvatarURL,
		Bio:       req.Bio,
	}

	a, err := application.UpdateAuthor(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toAuthorResponse(a))
}

func (h *AuthorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteAuthor(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthorHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreAuthor(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
