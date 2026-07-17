package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	attributedomain "github.com/trvux/elc-go/internal/attribute/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/product/application"
	"github.com/trvux/elc-go/internal/product/domain"
)

type ProductHandler struct {
	repo          domain.ProductRepository
	attributeRepo attributedomain.AttributeDefinitionRepository
}

func NewProductHandler(repo domain.ProductRepository, attributeRepo attributedomain.AttributeDefinitionRepository) *ProductHandler {
	return &ProductHandler{repo: repo, attributeRepo: attributeRepo}
}

func parseProductFilter(r *http.Request) (domain.ProductFilter, error) {
	q := r.URL.Query()

	filter := domain.ProductFilter{
		IncludeDeleted: q.Get("include_deleted") == "true",
	}

	if v := q.Get("category_id"); v != "" {
		filter.CategoryID = &v
	}
	if v := q.Get("category_ids"); v != "" {
		filter.CategoryIDs = splitNonEmpty(v)
	}
	if v := q.Get("brand_id"); v != "" {
		filter.BrandID = &v
	}
	if v := q.Get("brand_ids"); v != "" {
		filter.BrandIDs = splitNonEmpty(v)
	}
	if v := q.Get("product_line_id"); v != "" {
		filter.ProductLineID = &v
	}

	if v := q.Get("is_featured"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return filter, err
		}
		filter.IsFeatured = &b
	}
	if v := q.Get("is_published"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return filter, err
		}
		filter.IsPublished = &b
	}

	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return filter, err
		}
		filter.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return filter, err
		}
		filter.Offset = n
	}

	return filter, nil
}

func splitNonEmpty(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseProductFilter(r)
	if err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid query params", nil))
		return
	}

	result, err := application.ListProducts(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toProductListResponse(result))
}

func (h *ProductHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter, err := parseProductFilter(r)
	if err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid query params", nil))
		return
	}

	count, err := application.CountProducts(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := application.GetProductByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if p == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("product"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toProductResponse(p))
}

func (h *ProductHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	p, err := application.GetProductBySlug(r.Context(), h.repo, slug)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if p == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("product"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toProductResponse(p))
}

func (h *ProductHandler) GetByIDsBatch(w http.ResponseWriter, r *http.Request) {
	var req byIDsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	products, err := application.GetProductsByIDs(r.Context(), h.repo, req.IDs)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toProductResponseList(products))
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateProductInput{
		CategoryID: req.CategoryID, BrandID: req.BrandID,
		Name: req.Name, Slug: req.Slug,
		Description: req.Description,
		Images:      toImageAssetDomainList(req.Images), Labels: req.Labels,
		IsFeatured: req.IsFeatured, IsPublished: req.IsPublished, OrderIndex: req.OrderIndex,
		Condition: req.Condition,
		MetaTitle: req.MetaTitle, MetaDescription: req.MetaDescription,
		TagIDs:        req.TagIDs,
		ProductLineID: req.ProductLineID, ShortDescription: req.ShortDescription,
		WarrantyMonths: req.WarrantyMonths, WarrantyTerms: req.WarrantyTerms,
		Options:         toProductOptionInputList(req.Options),
		Variants:        toProductVariantInputList(req.Variants),
		AttributeValues: toAttributeValueInputList(req.AttributeValues),
	}

	p, err := application.CreateProduct(r.Context(), h.repo, h.attributeRepo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toPlainProductResponse(p))
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateProductInput{
		ID:         id,
		CategoryID: req.CategoryID, BrandID: req.BrandID,
		Name: req.Name, Slug: req.Slug,
		Description: req.Description,
		Images:      toImageAssetDomainList(req.Images), Labels: req.Labels,
		IsFeatured: req.IsFeatured, IsPublished: req.IsPublished, OrderIndex: req.OrderIndex,
		Condition: req.Condition,
		MetaTitle: req.MetaTitle, MetaDescription: req.MetaDescription,
		TagIDs:        req.TagIDs,
		ProductLineID: req.ProductLineID, ShortDescription: req.ShortDescription,
		WarrantyMonths: req.WarrantyMonths, WarrantyTerms: req.WarrantyTerms,
		Options:         toProductOptionInputListPtr(req.Options),
		Variants:        toProductVariantInputListPtr(req.Variants),
		AttributeValues: toAttributeValueInputListPtr(req.AttributeValues),
	}

	p, err := application.UpdateProduct(r.Context(), h.repo, h.attributeRepo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toPlainProductResponse(p))
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteProduct(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreProduct(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
