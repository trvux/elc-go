package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/catalog/domain"
)

type seoDTO struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Noindex     bool    `json:"noindex,omitempty"`
}

func toSeoDTO(seo domain.Seo) seoDTO {
	return seoDTO{Title: seo.Title, Description: seo.Description, Noindex: seo.Noindex}
}

func toSeoDomain(seo seoDTO) domain.Seo {
	return domain.Seo{Title: seo.Title, Description: seo.Description, Noindex: seo.Noindex}
}

type specSubItemDTO struct {
	Label string  `json:"label"`
	Value string  `json:"value"`
	Unit  *string `json:"unit,omitempty"`
}

type specItemDTO struct {
	Label string           `json:"label"`
	Value *string          `json:"value,omitempty"`
	Unit  *string          `json:"unit,omitempty"`
	Items []specSubItemDTO `json:"items,omitempty"`
}

func toSpecItemDTOList(specs []domain.SpecItem) []specItemDTO {
	if specs == nil {
		return nil
	}
	result := make([]specItemDTO, len(specs))
	for i, s := range specs {
		items := make([]specSubItemDTO, len(s.Items))
		for j, sub := range s.Items {
			items[j] = specSubItemDTO{Label: sub.Label, Value: sub.Value, Unit: sub.Unit}
		}
		result[i] = specItemDTO{Label: s.Label, Value: s.Value, Unit: s.Unit, Items: items}
	}
	return result
}

func toSpecItemDomainList(specs []specItemDTO) []domain.SpecItem {
	if specs == nil {
		return nil
	}
	result := make([]domain.SpecItem, len(specs))
	for i, s := range specs {
		items := make([]domain.SpecSubItem, len(s.Items))
		for j, sub := range s.Items {
			items[j] = domain.SpecSubItem{Label: sub.Label, Value: sub.Value, Unit: sub.Unit}
		}
		result[i] = domain.SpecItem{Label: s.Label, Value: s.Value, Unit: s.Unit, Items: items}
	}
	return result
}

type categoryRefResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
}

type brandRefResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	LogoURL         string  `json:"logo_url"`
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
	IsFeatured      bool    `json:"is_featured"`
	OrderIndex      int     `json:"order_index"`
}

// productResponse deliberately does NOT include normalized_specs — it's an
// internal write-time facet-indexing detail (see domain/spec_normalizer.go),
// never something any UI reads directly, same reasoning brand/service never
// expose derived state on their entities. `specs` here is always the
// ORIGINAL []SpecItem the client sent, not the derived facets.
type productResponse struct {
	ID              string               `json:"id"`
	CategoryID      string               `json:"category_id"`
	BrandID         string               `json:"brand_id"`
	Name            string               `json:"name"`
	SKU             string               `json:"sku"`
	Slug            string               `json:"slug"`
	Description     json.RawMessage      `json:"description"`
	Specs           []specItemDTO        `json:"specs"`
	Images          []string             `json:"images"`
	Labels          []string             `json:"labels"`
	OriginalPrice   int64                `json:"original_price"`
	SalePrice       *int64               `json:"sale_price"`
	DiscountPercent float64              `json:"discount_percent"`
	IsFeatured      bool                 `json:"is_featured"`
	IsPublished     bool                 `json:"is_published"`
	OrderIndex      int                  `json:"order_index"`
	StockStatus     string               `json:"stock_status"`
	Condition       string               `json:"condition"`
	MetaTitle       *string              `json:"meta_title"`
	MetaDescription *string              `json:"meta_description"`
	Seo             seoDTO               `json:"seo"`
	MPN             *string              `json:"mpn"`
	GTIN            *string              `json:"gtin"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	DeletedAt       *time.Time           `json:"deleted_at"`
	Category        *categoryRefResponse `json:"category"`
	Brand           *brandRefResponse    `json:"brand"`
}

func toProductResponse(p *domain.ProductWithRelations) productResponse {
	resp := toPlainProductResponse(p.Product)
	if p.Category != nil {
		resp.Category = &categoryRefResponse{
			ID: p.Category.ID, Name: p.Category.Name, Slug: p.Category.Slug,
			MetaTitle: p.Category.MetaTitle, MetaDescription: p.Category.MetaDescription,
		}
	}
	if p.Brand != nil {
		resp.Brand = &brandRefResponse{
			ID: p.Brand.ID, Name: p.Brand.Name, Slug: p.Brand.Slug, LogoURL: p.Brand.LogoURL,
			MetaTitle: p.Brand.MetaTitle, MetaDescription: p.Brand.MetaDescription,
			IsFeatured: p.Brand.IsFeatured, OrderIndex: p.Brand.OrderIndex,
		}
	}
	return resp
}

// toPlainProductResponse is used for Create/Update, which only ever return a
// plain *domain.Product (no joined relations) — same split as service's
// toPlainServiceResponse.
func toPlainProductResponse(p *domain.Product) productResponse {
	return productResponse{
		ID:              p.ID(),
		CategoryID:      p.CategoryID(),
		BrandID:         p.BrandID(),
		Name:            p.Name(),
		SKU:             p.SKU(),
		Slug:            p.Slug(),
		Description:     p.Description(),
		Specs:           toSpecItemDTOList(p.Specs()),
		Images:          p.Images(),
		Labels:          p.Labels(),
		OriginalPrice:   p.OriginalPrice(),
		SalePrice:       p.SalePrice(),
		DiscountPercent: p.DiscountPercent(),
		IsFeatured:      p.IsFeatured(),
		IsPublished:     p.IsPublished(),
		OrderIndex:      p.OrderIndex(),
		StockStatus:     p.StockStatus(),
		Condition:       p.Condition(),
		MetaTitle:       p.MetaTitle(),
		MetaDescription: p.MetaDescription(),
		Seo:             toSeoDTO(p.Seo()),
		MPN:             p.MPN(),
		GTIN:            p.GTIN(),
		CreatedAt:       p.CreatedAt(),
		UpdatedAt:       p.UpdatedAt(),
		DeletedAt:       p.DeletedAt(),
	}
}

func toProductResponseList(products []*domain.ProductWithRelations) []productResponse {
	result := make([]productResponse, len(products))
	for i, p := range products {
		result[i] = toProductResponse(p)
	}
	return result
}

type brandFacetResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type specFacetResponse struct {
	Label  string   `json:"label"`
	Values []string `json:"values"`
}

type facetsResponse struct {
	Brands   []brandFacetResponse `json:"brands"`
	Specs    []specFacetResponse  `json:"specs"`
	MinPrice int64                `json:"min_price"`
	MaxPrice int64                `json:"max_price"`
}

func toFacetsResponse(f domain.ProductFacets) facetsResponse {
	brands := make([]brandFacetResponse, len(f.Brands))
	for i, b := range f.Brands {
		brands[i] = brandFacetResponse{ID: b.ID, Name: b.Name, Slug: b.Slug}
	}
	specs := make([]specFacetResponse, len(f.Specs))
	for i, s := range f.Specs {
		specs[i] = specFacetResponse{Label: s.Label, Values: s.Values}
	}
	return facetsResponse{Brands: brands, Specs: specs, MinPrice: f.MinPrice, MaxPrice: f.MaxPrice}
}

type productListResponse struct {
	Data       []productResponse `json:"data"`
	TotalCount int               `json:"total_count"`
	Facets     facetsResponse    `json:"facets"`
}

func toProductListResponse(result *domain.ProductListResult) productListResponse {
	return productListResponse{
		Data:       toProductResponseList(result.Products),
		TotalCount: result.TotalCount,
		Facets:     toFacetsResponse(result.Facets),
	}
}

type adjacentProductResponse struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type adjacentProductsResponse struct {
	Prev *adjacentProductResponse `json:"prev"`
	Next *adjacentProductResponse `json:"next"`
}

func toAdjacentProductResponse(p *domain.AdjacentProduct) *adjacentProductResponse {
	if p == nil {
		return nil
	}
	return &adjacentProductResponse{Name: p.Name, Slug: p.Slug}
}

type createProductRequest struct {
	CategoryID      string          `json:"category_id"`
	BrandID         string          `json:"brand_id"`
	Name            string          `json:"name"`
	SKU             string          `json:"sku"`
	Slug            string          `json:"slug"`
	Description     json.RawMessage `json:"description"`
	Specs           []specItemDTO   `json:"specs"`
	Images          []string        `json:"images"`
	Labels          []string        `json:"labels"`
	OriginalPrice   int64           `json:"original_price"`
	SalePrice       *int64          `json:"sale_price"`
	DiscountPercent float64         `json:"discount_percent"`
	IsFeatured      bool            `json:"is_featured"`
	IsPublished     bool            `json:"is_published"`
	OrderIndex      int             `json:"order_index"`
	StockStatus     string          `json:"stock_status"`
	Condition       string          `json:"condition"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	Seo             seoDTO          `json:"seo"`
	MPN             *string         `json:"mpn"`
	GTIN            *string         `json:"gtin"`
}

type updateProductRequest struct {
	CategoryID      *string         `json:"category_id"`
	BrandID         *string         `json:"brand_id"`
	Name            *string         `json:"name"`
	SKU             *string         `json:"sku"`
	Slug            *string         `json:"slug"`
	Description     json.RawMessage `json:"description"`
	Specs           []specItemDTO   `json:"specs"`
	Images          []string        `json:"images"`
	Labels          []string        `json:"labels"`
	OriginalPrice   *int64          `json:"original_price"`
	SalePrice       *int64          `json:"sale_price"`
	DiscountPercent *float64        `json:"discount_percent"`
	IsFeatured      *bool           `json:"is_featured"`
	IsPublished     *bool           `json:"is_published"`
	OrderIndex      *int            `json:"order_index"`
	StockStatus     *string         `json:"stock_status"`
	Condition       *string         `json:"condition"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	Seo             *seoDTO         `json:"seo"`
	MPN             *string         `json:"mpn"`
	GTIN            *string         `json:"gtin"`
}

type byIDsRequest struct {
	IDs []string `json:"ids"`
}
