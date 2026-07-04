package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/service-group/application"
	"github.com/trvux/elc-go/internal/service-group/domain"
)

type ServiceGroupHandler struct {
	repo domain.ServiceGroupRepository
}

func NewServiceGroupHandler(repo domain.ServiceGroupRepository) *ServiceGroupHandler {
	return &ServiceGroupHandler{repo: repo}
}

func (h *ServiceGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.ServiceGroupFilter{
		IncludeDeleted: r.URL.Query().Get("include_deleted") == "true",
	}
	if raw := r.URL.Query().Get("is_featured"); raw != "" {
		isFeatured, err := strconv.ParseBool(raw)
		if err != nil {
			httpserver.WriteError(w, apperr.NewValidationError("invalid is_featured query param", nil))
			return
		}
		filter.IsFeatured = &isFeatured
	}

	groups, err := application.GetServiceGroups(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toServiceGroupResponseList(groups))
}

func (h *ServiceGroupHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	sg, err := application.GetServiceGroupByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if sg == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("service group"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toServiceGroupResponse(sg))
}

func (h *ServiceGroupHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	sg, err := application.GetServiceGroupBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if sg == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("service group"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toServiceGroupResponse(sg))
}

func (h *ServiceGroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createServiceGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateServiceGroupInput{
		Name:            req.Name,
		Slug:            req.Slug,
		ImageURL:        req.ImageURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		OrderIndex:      req.OrderIndex,
		CategoryIDs:     req.CategoryIDs,
	}

	sg, err := application.CreateServiceGroup(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toServiceGroupResponse(sg))
}

func (h *ServiceGroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateServiceGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateServiceGroupInput{
		ID:              id,
		Name:            req.Name,
		Slug:            req.Slug,
		ImageURL:        req.ImageURL,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		IsFeatured:      req.IsFeatured,
		OrderIndex:      req.OrderIndex,
		CategoryIDs:     req.CategoryIDs,
	}

	sg, err := application.UpdateServiceGroup(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toServiceGroupResponse(sg))
}

func (h *ServiceGroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteServiceGroup(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ServiceGroupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreServiceGroup(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
