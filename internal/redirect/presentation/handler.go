package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/redirect/application"
	"github.com/trvux/elc-go/internal/redirect/domain"
)

type Handler struct {
	repo domain.Repository
}

func NewHandler(repo domain.Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	items, total, err := application.ListRedirects(r.Context(), h.repo, domain.Filter{
		Search: q.Get("q"), Source: domain.Source(q.Get("source")), Limit: limit, Offset: offset,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	out := make([]redirectResponse, len(items))
	for i, it := range items {
		out[i] = toResponse(it)
	}
	httpserver.WriteJSON(w, http.StatusOK, listResponse{Items: out, Total: total})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}
	created, err := application.CreateRedirect(r.Context(), h.repo, domain.CreateRedirectInput{
		FromPath: req.FromPath, ToPath: req.ToPath, Note: req.Note,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, toResponse(created))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}
	updated, err := application.UpdateRedirect(r.Context(), h.repo, domain.UpdateRedirectInput{
		ID: chi.URLParam(r, "id"), ToPath: req.ToPath, Note: req.Note, IsActive: req.IsActive,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toResponse(updated))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := application.DeleteRedirect(r.Context(), h.repo, chi.URLParam(r, "id")); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
