package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/branch/application"
	"github.com/trvux/elc-go/internal/branch/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type BranchHandler struct {
	repo domain.BranchRepository
}

func NewBranchHandler(repo domain.BranchRepository) *BranchHandler {
	return &BranchHandler{repo: repo}
}

func (h *BranchHandler) List(w http.ResponseWriter, r *http.Request) {
	var isPublished *bool
	if raw := r.URL.Query().Get("is_published"); raw != "" {
		val := raw == "true"
		isPublished = &val
	}

	filter := domain.BranchFilter{
		IsPublished: isPublished,
		Search:      r.URL.Query().Get("search"),
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

	branches, err := application.GetBranches(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBranchResponseList(branches))
}

func (h *BranchHandler) Count(w http.ResponseWriter, r *http.Request) {
	var isPublished *bool
	if raw := r.URL.Query().Get("is_published"); raw != "" {
		val := raw == "true"
		isPublished = &val
	}

	filter := domain.BranchFilter{
		IsPublished: isPublished,
		Search:      r.URL.Query().Get("search"),
	}

	count, err := application.CountBranches(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *BranchHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	b, err := application.GetBranchByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if b == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("branch"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBranchResponse(b))
}

func (h *BranchHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	b, err := application.GetBranchBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if b == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("branch"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBranchResponse(b))
}

func (h *BranchHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateBranchInput{
		Name:            req.Name,
		Slug:            req.Slug,
		Address:         req.Address,
		Phone:           req.Phone,
		Email:           req.Email,
		MapsURL:         req.MapsURL,
		MapsEmbed:       req.MapsEmbed,
		ProvinceCode:    req.ProvinceCode,
		ProvinceName:    req.ProvinceName,
		WardCode:        req.WardCode,
		WardName:        req.WardName,
		PostalCode:      req.PostalCode,
		Description:     req.Description,
		Images:          toImageAssetDomainList(req.Images),
		IsPublished:     req.IsPublished,
		OrderIndex:      req.OrderIndex,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
	}

	b, err := application.CreateBranch(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toBranchResponse(b))
}

func (h *BranchHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateBranchInput{
		ID:              id,
		Name:            req.Name,
		Slug:            req.Slug,
		Address:         req.Address,
		Phone:           req.Phone,
		Email:           req.Email,
		MapsURL:         req.MapsURL,
		MapsEmbed:       req.MapsEmbed,
		ProvinceCode:    req.ProvinceCode,
		ProvinceName:    req.ProvinceName,
		WardCode:        req.WardCode,
		WardName:        req.WardName,
		PostalCode:      req.PostalCode,
		Description:     req.Description,
		Images:          toImageAssetDomainList(req.Images),
		IsPublished:     req.IsPublished,
		OrderIndex:      req.OrderIndex,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
	}

	b, err := application.UpdateBranch(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toBranchResponse(b))
}

func (h *BranchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteBranch(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BranchHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		OrderIndex int `json:"order_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if err := application.UpdateBranchOrder(r.Context(), h.repo, id, req.OrderIndex); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
