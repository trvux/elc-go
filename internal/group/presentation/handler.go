package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/group/application"
	"github.com/trvux/elc-go/internal/group/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type GroupHandler struct {
	repo domain.GroupRepository
}

func NewGroupHandler(repo domain.GroupRepository) *GroupHandler {
	return &GroupHandler{repo: repo}
}

func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.GroupFilter{
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

	groups, err := application.GetGroups(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toGroupResponseList(groups))
}

func (h *GroupHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	g, err := application.GetGroupByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if g == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("group"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toGroupResponse(g))
}

func (h *GroupHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	g, err := application.GetGroupBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if g == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("group"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toGroupResponse(g))
}

func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateGroupInput{
		Name:            req.Name,
		Slug:            req.Slug,
		ImageURL:        req.ImageURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		IsHidden:        req.IsHidden,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
	}

	g, err := application.CreateGroup(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toGroupResponse(g))
}

func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateGroupInput{
		ID:              id,
		Name:            req.Name,
		Slug:            req.Slug,
		ImageURL:        req.ImageURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		IsHidden:        req.IsHidden,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
	}

	g, err := application.UpdateGroup(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toGroupResponse(g))
}

func (h *GroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteGroup(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *GroupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreGroup(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
