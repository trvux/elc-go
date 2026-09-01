package presentation

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/project/application"
	"github.com/trvux/elc-go/internal/project/domain"
)

type ProjectHandler struct {
	repo domain.ProjectRepository
}

func NewProjectHandler(repo domain.ProjectRepository) *ProjectHandler {
	return &ProjectHandler{repo: repo}
}

func parseProjectFilter(r *http.Request) (domain.ProjectFilter, error) {
	q := r.URL.Query()
	filter := domain.ProjectFilter{
		Search:         q.Get("search"),
		IncludeDeleted: q.Get("include_deleted") == "true",
		OrderBy:        q.Get("order_by"),
		OrderDirection: q.Get("order_direction"),
	}
	if v := q.Get("project_type_id"); v != "" {
		filter.ProjectTypeID = &v
	}
	if v := q.Get("category_slug"); v != "" {
		filter.CategorySlug = &v
	}
	if v := q.Get("category_slugs"); v != "" {
		filter.CategorySlugs = strings.Split(v, ",")
	}
	if v := q.Get("service_slug"); v != "" {
		filter.ServiceSlug = &v
	}
	if v := q.Get("service_slugs"); v != "" {
		filter.ServiceSlugs = strings.Split(v, ",")
	}
	if v := q.Get("exclude_id"); v != "" {
		filter.ExcludeID = &v
	}
	if v := q.Get("is_published"); v != "" {
		b := v == "true"
		filter.IsPublished = &b
	}
	if v := q.Get("is_featured"); v != "" {
		b := v == "true"
		filter.IsFeatured = &b
	}
	limit, offset, err := httpserver.ParsePagination(r)
	if err != nil {
		return filter, err
	}
	filter.Limit = limit
	filter.Offset = offset
	return filter, nil
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseProjectFilter(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	projects, err := application.GetProjects(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toProjectResponseList(projects))
}

func (h *ProjectHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter, err := parseProjectFilter(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	count, err := application.CountProjects(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, countResponse{Count: count})
}

func (h *ProjectHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := application.GetProjectByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if p == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("project"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toProjectResponse(p))
}

func (h *ProjectHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	p, err := application.GetProjectBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if p == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("project"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toProjectResponse(p))
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateProjectInput{
		Title:             req.Title,
		Slug:              req.Slug,
		Description:       req.Description,
		Images:            toImageAssetDomainList(req.Images),
		IsFeatured:        req.IsFeatured,
		IsPublished:       req.IsPublished,
		MetaTitle:         req.MetaTitle,
		MetaDescription:   req.MetaDescription,
		OrderIndex:        req.OrderIndex,
		ProjectTypeID:     req.ProjectTypeID,
		ServiceIDs:        req.ServiceIDs,
		Categories:        toCategoryConditionDomainList(req.Categories),
		TagIDs:            req.TagIDs,
		ClientName:        req.ClientName,
		Location:          req.Location,
		CompletedAt:       req.CompletedAt,
		TestimonialQuote:  req.TestimonialQuote,
		TestimonialAuthor: req.TestimonialAuthor,
	}

	p, err := application.CreateProject(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toPlainProjectResponse(p))
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateProjectInput{
		ID:                id,
		Title:             req.Title,
		Slug:              req.Slug,
		Description:       req.Description,
		Images:            toImageAssetDomainList(req.Images),
		IsFeatured:        req.IsFeatured,
		IsPublished:       req.IsPublished,
		MetaTitle:         req.MetaTitle,
		MetaDescription:   req.MetaDescription,
		OrderIndex:        req.OrderIndex,
		ProjectTypeID:     req.ProjectTypeID,
		ServiceIDs:        req.ServiceIDs,
		Categories:        toCategoryConditionDomainPtr(req.Categories),
		TagIDs:            req.TagIDs,
		ClientName:        req.ClientName,
		Location:          req.Location,
		CompletedAt:       req.CompletedAt,
		TestimonialQuote:  req.TestimonialQuote,
		TestimonialAuthor: req.TestimonialAuthor,
	}

	p, err := application.UpdateProject(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toPlainProjectResponse(p))
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteProject(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreProject(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateOrderRequest struct {
	OrderIndex int `json:"order_index"`
}

func (h *ProjectHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if err := application.UpdateProjectOrder(r.Context(), h.repo, id, req.OrderIndex); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type toggleRequest struct {
	Value bool `json:"value"`
}

func (h *ProjectHandler) TogglePublish(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req toggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if err := application.ToggleProjectPublish(r.Context(), h.repo, id, req.Value); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) ToggleFeatured(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req toggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if err := application.ToggleProjectFeatured(r.Context(), h.repo, id, req.Value); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) GetAdjacent(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	currentID := q.Get("current_id")
	if currentID == "" {
		httpserver.WriteError(w, apperr.NewValidationError("current_id query param is required", nil))
		return
	}
	var projectTypeID *string
	if v := q.Get("project_type_id"); v != "" {
		projectTypeID = &v
	}

	prev, next, err := application.GetAdjacentProjects(r.Context(), h.repo, projectTypeID, currentID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, adjacentProjectsResponse{
		Prev: toAdjacentProjectResponse(prev),
		Next: toAdjacentProjectResponse(next),
	})
}

func (h *ProjectHandler) GetCategoriesByProjectType(w http.ResponseWriter, r *http.Request) {
	projectTypeID := chi.URLParam(r, "projectTypeId")

	categories, err := application.GetCategoriesByProjectTypeID(r.Context(), h.repo, projectTypeID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toCategoryRefResponseList(categories))
}
