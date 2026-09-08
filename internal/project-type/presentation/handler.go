package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/project-type/application"
	"github.com/trvux/elc-go/internal/project-type/domain"
)

type ProjectTypeHandler struct {
	repo domain.ProjectTypeRepository
}

func NewProjectTypeHandler(repo domain.ProjectTypeRepository) *ProjectTypeHandler {
	return &ProjectTypeHandler{repo: repo}
}

func filterFromQuery(r *http.Request) (domain.ProjectTypeFilter, error) {
	filter := domain.ProjectTypeFilter{
		Search:         r.URL.Query().Get("search"),
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

func (h *ProjectTypeHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := filterFromQuery(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	projectTypes, err := application.GetProjectTypes(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toProjectTypeResponseList(projectTypes))
}

func (h *ProjectTypeHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter, err := filterFromQuery(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	count, err := application.CountProjectTypes(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *ProjectTypeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	pt, err := application.GetProjectTypeByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if pt == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("project type"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toProjectTypeResponse(pt))
}

func (h *ProjectTypeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProjectTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateProjectTypeInput{
		Name:            req.Name,
		Slug:            req.Slug,
		Image:           req.Image,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
		CategoryIDs:     req.CategoryIDs,
	}

	pt, err := application.CreateProjectType(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toBareProjectTypeResponse(pt))
}

func (h *ProjectTypeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateProjectTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateProjectTypeInput{
		ID:              id,
		Name:            req.Name,
		Slug:            req.Slug,
		Image:           req.Image,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		OrderIndex:      req.OrderIndex,
		Content:         req.Content,
		CategoryIDs:     req.CategoryIDs,
	}

	pt, err := application.UpdateProjectType(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBareProjectTypeResponse(pt))
}

func (h *ProjectTypeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteProjectType(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
