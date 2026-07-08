package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/catalog/application"
	"github.com/trvux/elc-go/internal/catalog/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type ProductHandler struct {
	repo domain.ProductRepository
}

func NewProductHandler(repo domain.ProductRepository) *ProductHandler {
	return &ProductHandler{repo: repo}
}

func parseProductFilter(r *http.Request) (domain.ProductFilter, error) {
	q := r.URL.Query()

	filter := domain.ProductFilter{
		Search:         q.Get("search"),
		SortBy:         q.Get("sort_by"),
		Condition:      q.Get("condition"),
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
	if v := q.Get("brand_slugs"); v != "" {
		filter.BrandSlugs = splitNonEmpty(v)
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

	if v := q.Get("min_price"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return filter, err
		}
		filter.MinPrice = &n
	}
	if v := q.Get("max_price"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return filter, err
		}
		filter.MaxPrice = &n
	}

	// spec_<label>=<value> query params, repeatable per label — mirrors the
	// existing elc-tem UI convention. Go's net/url already percent-decodes
	// both keys and values while parsing the query string, so the Vietnamese
	// UI labels in the key (e.g. "spec_C%C3%B4ng%20su%E1%BA%A5t") come out
	// correctly decoded with no extra handling needed here.
	specs := map[string][]string{}
	for key, values := range q {
		if label, ok := strings.CutPrefix(key, "spec_"); ok && label != "" {
			specs[label] = append(specs[label], values...)
		}
	}
	if len(specs) > 0 {
		filter.Specs = specs
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

func (h *ProductHandler) GetAdjacent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	current, err := application.GetProductByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if current == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("product"))
		return
	}

	prev, next, err := application.GetAdjacentProducts(r.Context(), h.repo, current.CategoryID(), current.ID())
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, adjacentProductsResponse{
		Prev: toAdjacentProductResponse(prev),
		Next: toAdjacentProductResponse(next),
	})
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
		Name: req.Name, SKU: req.SKU, Slug: req.Slug,
		Description: req.Description, Specs: toSpecItemDomainList(req.Specs),
		Images: toImageAssetDomainList(req.Images), Labels: req.Labels,
		OriginalPrice: req.OriginalPrice, SalePrice: req.SalePrice, DiscountPercent: req.DiscountPercent,
		IsFeatured: req.IsFeatured, IsPublished: req.IsPublished, OrderIndex: req.OrderIndex,
		StockStatus: req.StockStatus, Condition: req.Condition,
		MetaTitle: req.MetaTitle, MetaDescription: req.MetaDescription, Seo: toSeoDomain(req.Seo),
		MPN: req.MPN, GTIN: req.GTIN, TagIDs: req.TagIDs,
	}

	p, err := application.CreateProduct(r.Context(), h.repo, input)
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

	var seo *domain.Seo
	if req.Seo != nil {
		s := toSeoDomain(*req.Seo)
		seo = &s
	}

	input := domain.UpdateProductInput{
		ID:         id,
		CategoryID: req.CategoryID, BrandID: req.BrandID,
		Name: req.Name, SKU: req.SKU, Slug: req.Slug,
		Description: req.Description, Specs: toSpecItemDomainList(req.Specs),
		Images: toImageAssetDomainList(req.Images), Labels: req.Labels,
		OriginalPrice: req.OriginalPrice, SalePrice: req.SalePrice, DiscountPercent: req.DiscountPercent,
		IsFeatured: req.IsFeatured, IsPublished: req.IsPublished, OrderIndex: req.OrderIndex,
		StockStatus: req.StockStatus, Condition: req.Condition,
		MetaTitle: req.MetaTitle, MetaDescription: req.MetaDescription, Seo: seo,
		MPN: req.MPN, GTIN: req.GTIN, TagIDs: req.TagIDs,
	}

	p, err := application.UpdateProduct(r.Context(), h.repo, input)
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
