package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/product/application"
	"github.com/trvux/elc-go/internal/product/domain"
)

// ProductLineHandler is deliberately a separate handler (own struct, own
// repository) from ProductHandler — ProductLine is a simple independent
// CRUD lookup, same shape as brand/category's handlers, not something that
// needs Product's richer filter/facet machinery.
type ProductLineHandler struct {
	repo domain.ProductLineRepository
}

func NewProductLineHandler(repo domain.ProductLineRepository) *ProductLineHandler {
	return &ProductLineHandler{repo: repo}
}

func (h *ProductLineHandler) List(w http.ResponseWriter, r *http.Request) {
	var brandID *string
	if v := r.URL.Query().Get("brand_id"); v != "" {
		brandID = &v
	}
	includeDeleted := r.URL.Query().Get("include_deleted") == "true"

	lines, err := application.ListProductLines(r.Context(), h.repo, brandID, includeDeleted)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toProductLineResponseList(lines))
}

func (h *ProductLineHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	l, err := application.GetProductLine(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if l == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("product line"))
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toProductLineResponse(l))
}

func (h *ProductLineHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProductLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	l, err := application.CreateProductLine(r.Context(), h.repo, domain.CreateProductLineInput{
		BrandID: req.BrandID, CategoryID: req.CategoryID, Code: req.Code, Name: req.Name,
		TierRank: req.TierRank, Description: req.Description, MpnPrefixes: req.MpnPrefixes,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, toProductLineResponse(l))
}

func (h *ProductLineHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateProductLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	l, err := application.UpdateProductLine(r.Context(), h.repo, domain.UpdateProductLineInput{
		ID: id, CategoryID: req.CategoryID, Name: req.Name, TierRank: req.TierRank,
		Description: req.Description, MpnPrefixes: req.MpnPrefixes,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toProductLineResponse(l))
}

func (h *ProductLineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := application.DeleteProductLine(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductLineHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := application.RestoreProductLine(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterProductLineRoutes(r chi.Router, h *ProductLineHandler, verifier httpserver.TokenVerifier) {
	r.Route("/product-lines", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Post("/", h.Create)
			r.Put("/{id}", h.Update)
			r.Post("/{id}/restore", h.Restore)
		})
		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanDeleteContent))
			r.Delete("/{id}", h.Delete)
		})
	})
}
