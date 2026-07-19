package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/product/domain"
)

type imageAssetDTO struct {
	URL     string `json:"url"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
}

func toImageAssetDTOList(images []domain.ImageAsset) []imageAssetDTO {
	result := make([]imageAssetDTO, len(images))
	for i, img := range images {
		result[i] = imageAssetDTO{URL: img.URL, Alt: img.Alt, Caption: img.Caption}
	}
	return result
}

func toImageAssetDomainList(dtos []imageAssetDTO) []domain.ImageAsset {
	if dtos == nil {
		return nil
	}
	result := make([]domain.ImageAsset, len(dtos))
	for i, d := range dtos {
		result[i] = domain.ImageAsset{URL: d.URL, Alt: d.Alt, Caption: d.Caption}
	}
	return result
}

type tagRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func toTagRefResponseList(tags []domain.TagRef) []tagRefResponse {
	result := make([]tagRefResponse, len(tags))
	for i, t := range tags {
		result[i] = tagRefResponse{ID: t.ID, Name: t.Name, Slug: t.Slug}
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

type productResponse struct {
	ID                 string                   `json:"id"`
	CategoryID         string                   `json:"category_id"`
	BrandID            string                   `json:"brand_id"`
	Name               string                   `json:"name"`
	Slug               string                   `json:"slug"`
	Description        json.RawMessage          `json:"description"`
	Images             []imageAssetDTO          `json:"images"`
	IsFeatured         bool                     `json:"is_featured"`
	Status             string                   `json:"status"`
	RejectionReason    *string                  `json:"rejection_reason"`
	OrderIndex         int                      `json:"order_index"`
	MetaTitle          *string                  `json:"meta_title"`
	MetaDescription    *string                  `json:"meta_description"`
	ProductLineID      *string                  `json:"product_line_id"`
	ShortDescription   *string                  `json:"short_description"`
	DefaultVariantID   *string                  `json:"default_variant_id"`
	DisplayPrice       *int64                   `json:"display_price"`
	DisplayStockStatus *string                  `json:"display_stock_status"`
	PriceMin           *int64                   `json:"price_min"`
	PriceMax           *int64                   `json:"price_max"`
	CreatedAt          time.Time                `json:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
	DeletedAt          *time.Time               `json:"deleted_at"`
	Category           *categoryRefResponse     `json:"category"`
	Brand              *brandRefResponse        `json:"brand"`
	Tags               []tagRefResponse         `json:"tags"`
	Options            []productOptionResponse  `json:"options,omitempty"`
	Variants           []productVariantResponse `json:"variants,omitempty"`
	AttributeValues    []attributeValueResponse `json:"attribute_values,omitempty"`
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
	resp.Tags = toTagRefResponseList(p.Tags)
	if p.Options != nil {
		resp.Options = toProductOptionResponseList(p.Options)
	}
	if p.Variants != nil {
		resp.Variants = toProductVariantResponseList(p.Variants)
	}
	if p.AttributeValues != nil {
		resp.AttributeValues = toAttributeValueResponseList(p.AttributeValues)
	}
	return resp
}

// toPlainProductResponse is used for Create/Update, which only ever return a
// plain *domain.Product (no joined relations) — same split as service's
// toPlainServiceResponse.
func toPlainProductResponse(p *domain.Product) productResponse {
	return productResponse{
		ID:                 p.ID(),
		CategoryID:         p.CategoryID(),
		BrandID:            p.BrandID(),
		Name:               p.Name(),
		Slug:               p.Slug(),
		Description:        p.Description(),
		Images:             toImageAssetDTOList(p.Images()),
		IsFeatured:         p.IsFeatured(),
		Status:             string(p.Status()),
		RejectionReason:    p.RejectionReason(),
		OrderIndex:         p.OrderIndex(),
		MetaTitle:          p.MetaTitle(),
		MetaDescription:    p.MetaDescription(),
		ProductLineID:      p.ProductLineID(),
		ShortDescription:   p.ShortDescription(),
		DefaultVariantID:   p.DefaultVariantID(),
		DisplayPrice:       p.DisplayPrice(),
		DisplayStockStatus: p.DisplayStockStatus(),
		PriceMin:           p.PriceMin(),
		PriceMax:           p.PriceMax(),
		CreatedAt:          p.CreatedAt(),
		UpdatedAt:          p.UpdatedAt(),
		DeletedAt:          p.DeletedAt(),
	}
}

func toProductResponseList(products []*domain.ProductWithRelations) []productResponse {
	result := make([]productResponse, len(products))
	for i, p := range products {
		result[i] = toProductResponse(p)
	}
	return result
}

type productListResponse struct {
	Data       []productResponse     `json:"data"`
	TotalCount int                   `json:"total_count"`
	Facets     productFacetsResponse `json:"facets"`
}

func toProductListResponse(result *domain.ProductListResult) productListResponse {
	return productListResponse{
		Data:       toProductResponseList(result.Products),
		TotalCount: result.TotalCount,
		Facets:     toProductFacetsResponse(result.Facets),
	}
}

type brandFacetResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	LogoURL string `json:"logo_url"`
	Count   int    `json:"count"`
}

type numberBucketResponse struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int     `json:"count"`
}

func toNumberBucketResponseList(buckets []domain.NumberBucket) []numberBucketResponse {
	result := make([]numberBucketResponse, len(buckets))
	for i, b := range buckets {
		result[i] = numberBucketResponse{Min: b.Min, Max: b.Max, Count: b.Count}
	}
	return result
}

type priceFacetResponse struct {
	Min     int64                  `json:"min"`
	Max     int64                  `json:"max"`
	Buckets []numberBucketResponse `json:"buckets"`
}

type attributeFacetOptionResponse struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type attributeFacetResponse struct {
	Code       string                         `json:"code"`
	Name       string                         `json:"name"`
	GroupLabel *string                        `json:"group_label"`
	DataType   string                         `json:"data_type"`
	Unit       *string                        `json:"unit"`
	Options    []attributeFacetOptionResponse `json:"options,omitempty"`
	Min        *float64                       `json:"min,omitempty"`
	Max        *float64                       `json:"max,omitempty"`
	Buckets    []numberBucketResponse         `json:"buckets,omitempty"`
}

type productFacetsResponse struct {
	Brands     []brandFacetResponse     `json:"brands"`
	Price      priceFacetResponse       `json:"price"`
	Attributes []attributeFacetResponse `json:"attributes"`
}

func toProductFacetsResponse(facets domain.ProductFacets) productFacetsResponse {
	brands := make([]brandFacetResponse, len(facets.Brands))
	for i, b := range facets.Brands {
		brands[i] = brandFacetResponse{ID: b.ID, Name: b.Name, Slug: b.Slug, LogoURL: b.LogoURL, Count: b.Count}
	}

	attributes := make([]attributeFacetResponse, len(facets.Attributes))
	for i, a := range facets.Attributes {
		options := make([]attributeFacetOptionResponse, len(a.Options))
		for j, o := range a.Options {
			options[j] = attributeFacetOptionResponse{Value: o.Value, Count: o.Count}
		}
		attributes[i] = attributeFacetResponse{
			Code: a.Code, Name: a.Name, GroupLabel: a.GroupLabel, DataType: a.DataType, Unit: a.Unit,
			Options: options, Min: a.Min, Max: a.Max, Buckets: toNumberBucketResponseList(a.Buckets),
		}
	}

	return productFacetsResponse{
		Brands: brands,
		Price: priceFacetResponse{
			Min: facets.Price.Min, Max: facets.Price.Max,
			Buckets: toNumberBucketResponseList(facets.Price.Buckets),
		},
		Attributes: attributes,
	}
}

type productOptionValueResponse struct {
	ID         string `json:"id"`
	Value      string `json:"value"`
	OrderIndex int    `json:"order_index"`
}

type productOptionResponse struct {
	ID         string                       `json:"id"`
	Name       string                       `json:"name"`
	OrderIndex int                          `json:"order_index"`
	Values     []productOptionValueResponse `json:"values"`
}

func toProductOptionResponseList(options []domain.ProductOption) []productOptionResponse {
	result := make([]productOptionResponse, len(options))
	for i, o := range options {
		values := make([]productOptionValueResponse, len(o.Values))
		for j, v := range o.Values {
			values[j] = productOptionValueResponse{ID: v.ID, Value: v.Value, OrderIndex: v.OrderIndex}
		}
		result[i] = productOptionResponse{ID: o.ID, Name: o.Name, OrderIndex: o.OrderIndex, Values: values}
	}
	return result
}

type variantComponentResponse struct {
	ID           string  `json:"id"`
	ComponentID  string  `json:"component_id"`
	ComponentMPN string  `json:"component_mpn"`
	ComponentSKU string  `json:"component_sku"`
	Quantity     int     `json:"quantity"`
	Role         *string `json:"role,omitempty"`
}

// productVariantResponse deliberately omits cost_price — internal margin
// data, never exposed through the API even though it's stored, per
// docs/product-v2-design.md.
type productVariantResponse struct {
	ID              string                     `json:"id"`
	MPN             string                     `json:"mpn"`
	SKU             string                     `json:"sku"`
	GTIN            *string                    `json:"gtin"`
	IsDefault       bool                       `json:"is_default"`
	IsStandalone    bool                       `json:"is_standalone"`
	StockStatus     string                     `json:"stock_status"`
	LeadTimeDays    *int                       `json:"lead_time_days"`
	OriginalPrice   int64                      `json:"original_price"`
	SalePrice       *int64                     `json:"sale_price"`
	DiscountPercent float64                    `json:"discount_percent"`
	DisplayPrice    int64                      `json:"display_price"`
	Weight          *float64                   `json:"weight"`
	IsActive        bool                       `json:"is_active"`
	OrderIndex      int                        `json:"order_index"`
	OptionValueIDs  []string                   `json:"option_value_ids"`
	Components      []variantComponentResponse `json:"components,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

func toProductVariantResponseList(variants []*domain.ProductVariant) []productVariantResponse {
	result := make([]productVariantResponse, len(variants))
	for i, v := range variants {
		components := make([]variantComponentResponse, len(v.Components()))
		for j, c := range v.Components() {
			components[j] = variantComponentResponse{
				ID: c.ID, ComponentID: c.Component.ID, ComponentMPN: c.Component.MPN, ComponentSKU: c.Component.SKU,
				Quantity: c.Quantity, Role: c.Role,
			}
		}
		result[i] = productVariantResponse{
			ID: v.ID(), MPN: v.MPN(), SKU: v.SKU(), GTIN: v.GTIN(),
			IsDefault: v.IsDefault(), IsStandalone: v.IsStandalone(), StockStatus: v.StockStatus(),
			LeadTimeDays:  v.LeadTimeDays(),
			OriginalPrice: v.OriginalPrice(), SalePrice: v.SalePrice(), DiscountPercent: v.DiscountPercent(),
			DisplayPrice: v.DisplayPrice(), Weight: v.Weight(), IsActive: v.IsActive(), OrderIndex: v.OrderIndex(),
			OptionValueIDs: v.OptionValueIDs(), Components: components,
			CreatedAt: v.CreatedAt(), UpdatedAt: v.UpdatedAt(),
		}
	}
	return result
}

type variantOptionSelectionDTO struct {
	OptionName string `json:"option_name"`
	Value      string `json:"value"`
}

func toVariantOptionSelectionDomainList(dtos []variantOptionSelectionDTO) []domain.VariantOptionSelection {
	result := make([]domain.VariantOptionSelection, len(dtos))
	for i, d := range dtos {
		result[i] = domain.VariantOptionSelection{OptionName: d.OptionName, Value: d.Value}
	}
	return result
}

type variantComponentRequestDTO struct {
	ComponentIndex int     `json:"component_index"`
	Quantity       int     `json:"quantity"`
	Role           *string `json:"role,omitempty"`
}

func toVariantComponentDomainList(dtos []variantComponentRequestDTO) []domain.VariantComponentInput {
	result := make([]domain.VariantComponentInput, len(dtos))
	for i, d := range dtos {
		result[i] = domain.VariantComponentInput{ComponentIndex: d.ComponentIndex, Quantity: d.Quantity, Role: d.Role}
	}
	return result
}

type productOptionRequestDTO struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

func toProductOptionInputList(dtos []productOptionRequestDTO) []domain.ProductOptionInput {
	result := make([]domain.ProductOptionInput, len(dtos))
	for i, d := range dtos {
		result[i] = domain.ProductOptionInput{Name: d.Name, Values: d.Values}
	}
	return result
}

type productVariantRequestDTO struct {
	MPN              string                       `json:"mpn"`
	SKU              *string                      `json:"sku,omitempty"`
	GTIN             *string                      `json:"gtin,omitempty"`
	IsDefault        bool                         `json:"is_default"`
	IsComponentOnly  bool                         `json:"is_component_only"`
	StockStatus      string                       `json:"stock_status"`
	LeadTimeDays     *int                         `json:"lead_time_days,omitempty"`
	CostPrice        *int64                       `json:"cost_price,omitempty"`
	OriginalPrice    int64                        `json:"original_price"`
	SalePrice        *int64                       `json:"sale_price,omitempty"`
	DiscountPercent  float64                      `json:"discount_percent"`
	Weight           *float64                     `json:"weight,omitempty"`
	IsActive         bool                         `json:"is_active"`
	OrderIndex       int                          `json:"order_index"`
	OptionSelections []variantOptionSelectionDTO  `json:"option_selections,omitempty"`
	Components       []variantComponentRequestDTO `json:"components,omitempty"`
}

func toProductVariantInputList(dtos []productVariantRequestDTO) []domain.ProductVariantInput {
	result := make([]domain.ProductVariantInput, len(dtos))
	for i, d := range dtos {
		result[i] = domain.ProductVariantInput{
			MPN: d.MPN, SKU: d.SKU, GTIN: d.GTIN, IsDefault: d.IsDefault, IsComponentOnly: d.IsComponentOnly,
			StockStatus: d.StockStatus, LeadTimeDays: d.LeadTimeDays, CostPrice: d.CostPrice,
			OriginalPrice: d.OriginalPrice, SalePrice: d.SalePrice, DiscountPercent: d.DiscountPercent,
			Weight: d.Weight, IsActive: d.IsActive, OrderIndex: d.OrderIndex,
			OptionSelections: toVariantOptionSelectionDomainList(d.OptionSelections),
			Components:       toVariantComponentDomainList(d.Components),
		}
	}
	return result
}

func toProductOptionInputListPtr(dtos *[]productOptionRequestDTO) *[]domain.ProductOptionInput {
	if dtos == nil {
		return nil
	}
	result := toProductOptionInputList(*dtos)
	return &result
}

func toProductVariantInputListPtr(dtos *[]productVariantRequestDTO) *[]domain.ProductVariantInput {
	if dtos == nil {
		return nil
	}
	result := toProductVariantInputList(*dtos)
	return &result
}

type attributeValueRequestDTO struct {
	AttributeDefinitionID string   `json:"attribute_definition_id"`
	ValueText             *string  `json:"value_text,omitempty"`
	ValueNumber           *float64 `json:"value_number,omitempty"`
	ValueBoolean          *bool    `json:"value_boolean,omitempty"`
	ValueOptions          []string `json:"value_options,omitempty"`
}

func toAttributeValueInputList(dtos []attributeValueRequestDTO) []domain.ProductAttributeValueInput {
	result := make([]domain.ProductAttributeValueInput, len(dtos))
	for i, d := range dtos {
		result[i] = domain.ProductAttributeValueInput{
			AttributeDefinitionID: d.AttributeDefinitionID,
			ValueText:             d.ValueText, ValueNumber: d.ValueNumber, ValueBoolean: d.ValueBoolean,
			ValueOptions: d.ValueOptions,
		}
	}
	return result
}

func toAttributeValueInputListPtr(dtos *[]attributeValueRequestDTO) *[]domain.ProductAttributeValueInput {
	if dtos == nil {
		return nil
	}
	result := toAttributeValueInputList(*dtos)
	return &result
}

type attributeValueResponse struct {
	ID                    string   `json:"id"`
	AttributeDefinitionID string   `json:"attribute_definition_id"`
	Code                  string   `json:"code"`
	Name                  string   `json:"name"`
	GroupLabel            *string  `json:"group_label"`
	DataType              string   `json:"data_type"`
	Unit                  *string  `json:"unit"`
	Options               []string `json:"options"`
	ValueText             *string  `json:"value_text"`
	ValueNumber           *float64 `json:"value_number"`
	ValueBoolean          *bool    `json:"value_boolean"`
	ValueOptions          []string `json:"value_options"`
}

func toAttributeValueResponseList(refs []domain.AttributeValueRef) []attributeValueResponse {
	result := make([]attributeValueResponse, len(refs))
	for i, ref := range refs {
		result[i] = attributeValueResponse{
			ID: ref.ID, AttributeDefinitionID: ref.AttributeDefinitionID, Code: ref.Code, Name: ref.Name,
			GroupLabel: ref.GroupLabel, DataType: ref.DataType, Unit: ref.Unit, Options: ref.Options,
			ValueText: ref.ValueText, ValueNumber: ref.ValueNumber, ValueBoolean: ref.ValueBoolean,
			ValueOptions: ref.ValueOptions,
		}
	}
	return result
}

// createProductRequest deliberately carries no status field — a created
// product always starts as draft, see domain.CreateProductInput's doc
// comment.
type createProductRequest struct {
	CategoryID       string                    `json:"category_id"`
	BrandID          string                    `json:"brand_id"`
	Name             string                    `json:"name"`
	Slug             string                    `json:"slug"`
	Description      json.RawMessage           `json:"description"`
	Images           []imageAssetDTO           `json:"images"`
	IsFeatured       bool                      `json:"is_featured"`
	OrderIndex       int                       `json:"order_index"`
	MetaTitle        *string                   `json:"meta_title"`
	MetaDescription  *string                   `json:"meta_description"`
	TagIDs           []string                  `json:"tag_ids"`
	ProductLineID    *string                   `json:"product_line_id,omitempty"`
	ShortDescription *string                   `json:"short_description,omitempty"`
	Options          []productOptionRequestDTO `json:"options,omitempty"`
	// Variants must contain at least one entry — see domain.CreateProductInput.
	Variants        []productVariantRequestDTO `json:"variants,omitempty"`
	AttributeValues []attributeValueRequestDTO `json:"attribute_values,omitempty"`
}

// updateProductRequest deliberately carries no status field — status only
// moves through the dedicated submit/approve/reject/archive endpoints, see
// domain.UpdateProductInput's doc comment.
type updateProductRequest struct {
	CategoryID       *string         `json:"category_id"`
	BrandID          *string         `json:"brand_id"`
	Name             *string         `json:"name"`
	Slug             *string         `json:"slug"`
	Description      json.RawMessage `json:"description"`
	Images           []imageAssetDTO `json:"images"`
	IsFeatured       *bool           `json:"is_featured"`
	OrderIndex       *int            `json:"order_index"`
	MetaTitle        *string         `json:"meta_title"`
	MetaDescription  *string         `json:"meta_description"`
	TagIDs           *[]string       `json:"tag_ids"`
	ProductLineID    *string         `json:"product_line_id"`
	ShortDescription *string         `json:"short_description"`
	// Options/Variants: absent from the JSON body (nil) leaves the variant
	// tree untouched; present (even as []) replaces it wholesale — same
	// convention as domain.UpdateProductInput. A present Variants must
	// contain at least one entry.
	Options  *[]productOptionRequestDTO  `json:"options"`
	Variants *[]productVariantRequestDTO `json:"variants"`
	// AttributeValues: nil = leave untouched, non-nil (even []) = replace
	// wholesale — same convention as Options/Variants.
	AttributeValues *[]attributeValueRequestDTO `json:"attribute_values"`
}

type byIDsRequest struct {
	IDs []string `json:"ids"`
}
