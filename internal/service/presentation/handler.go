package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/service/application"
	"github.com/trvux/elc-go/internal/service/domain"
)

type ServiceHandler struct {
	repo domain.ServiceRepository
}

func NewServiceHandler(repo domain.ServiceRepository) *ServiceHandler {
	return &ServiceHandler{repo: repo}
}

func parseServiceFilter(r *http.Request) (domain.ServiceFilter, error) {
	filter := domain.ServiceFilter{
		Search:         r.URL.Query().Get("search"),
		IncludeDeleted: r.URL.Query().Get("include_deleted") == "true",
	}
	if v := r.URL.Query().Get("group_id"); v != "" {
		filter.GroupID = &v
	}
	if v := r.URL.Query().Get("category_id"); v != "" {
		filter.CategoryID = &v
	}
	if v := r.URL.Query().Get("is_featured"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return filter, err
		}
		filter.IsFeatured = &b
	}
	if v := r.URL.Query().Get("is_published"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return filter, err
		}
		filter.IsPublished = &b
	}
	return filter, nil
}

func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseServiceFilter(r)
	if err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid query params", nil))
		return
	}

	services, err := application.GetServices(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toServiceResponseList(services))
}

func (h *ServiceHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter, err := parseServiceFilter(r)
	if err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid query params", nil))
		return
	}

	count, err := application.CountServices(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *ServiceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s, err := application.GetServiceByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if s == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("service"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toServiceResponse(s))
}

func (h *ServiceHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	s, err := application.GetServiceBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if s == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("service"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toServiceResponse(s))
}

func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateServiceInput{
		Title: req.Title, Slug: req.Slug,
		GroupID: req.GroupID, CategoryID: req.CategoryID,
		OriginalPrice: req.OriginalPrice, DiscountPercent: req.DiscountPercent,
		PriceDisplayText: req.PriceDisplayText, Labels: req.Labels,
		Description: req.Description, Content: req.Content,
		Image: req.Image, MetaTitle: req.MetaTitle, MetaDescription: req.MetaDescription,
		Seo:        toSeoDomain(req.Seo),
		IsFeatured: req.IsFeatured, IsPublished: req.IsPublished, OrderIndex: req.OrderIndex,
	}

	s, err := application.CreateService(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toPlainServiceResponse(s))
}

func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	var seo *domain.Seo
	if req.Seo != nil {
		s := toSeoDomain(*req.Seo)
		seo = &s
	}

	input := domain.UpdateServiceInput{
		ID: id, Title: req.Title, Slug: req.Slug,
		GroupID: req.GroupID, CategoryID: req.CategoryID,
		OriginalPrice: req.OriginalPrice, DiscountPercent: req.DiscountPercent,
		PriceDisplayText: req.PriceDisplayText, Labels: req.Labels,
		Description: req.Description, Content: req.Content,
		Image: req.Image, MetaTitle: req.MetaTitle, MetaDescription: req.MetaDescription,
		Seo:        seo,
		IsFeatured: req.IsFeatured, IsPublished: req.IsPublished, OrderIndex: req.OrderIndex,
	}

	s, err := application.UpdateService(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toPlainServiceResponse(s))
}

func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteService(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ServiceHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreService(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
