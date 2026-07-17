package domain

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/media"
)

// ImageAsset re-exports the shared media type so callers outside this
// package can write domain.ImageAsset without also importing
// internal/platform/media directly.
type ImageAsset = media.ImageAsset

// SpecSubItem/SpecItem model the jsonb shape stored in products.specs — ported
// verbatim from elc-tem's modules/catalog/domain/types.ts (SpecSubItem/SpecItem).
// Value is a plain string on SpecSubItem (always present in the old TS type)
// but a pointer on SpecItem (optional — a SpecItem either carries a single
// Value or a nested Items list, never both in practice, mirroring the TS
// `value?: string` / `items?: SpecSubItem[]`).
type SpecSubItem struct {
	Label string  `json:"label"`
	Value string  `json:"value"`
	Unit  *string `json:"unit,omitempty"`
}

type SpecItem struct {
	Label string        `json:"label"`
	Value *string       `json:"value,omitempty"`
	Unit  *string       `json:"unit,omitempty"`
	Items []SpecSubItem `json:"items,omitempty"`
}

// CategoryRef/BrandRef are lightweight, read-only references to entities
// owned by other modules (category, brand) — deliberately NOT the full
// category/brand.Brand domain types, same reasoning as service's
// GroupRef/CategoryRef (see internal/service/domain/types.go): avoids a
// cross-module Go dependency for the handful of fields any UI actually
// reads from a join. category/brand columns are SELECTed directly in this
// module's own SQL (internal/product/infrastructure) rather than importing
// internal/brand.
type CategoryRef struct {
	ID              string
	Name            string
	Slug            string
	MetaTitle       *string
	MetaDescription *string
}

type BrandRef struct {
	ID              string
	Name            string
	Slug            string
	LogoURL         string
	MetaTitle       *string
	MetaDescription *string
	IsFeatured      bool
	OrderIndex      int
}

// TagRef is a lightweight read-only reference to a tag owned by the tag
// module — resolved via a direct SQL join into `tags`/`product_tags`, same
// cross-module read pattern as CategoryRef/BrandRef above.
type TagRef struct {
	ID   string
	Name string
	Slug string
}

// Seo is the unified SEO metadata shape stored as jsonb on products/news/
// projects (see docs/catalog.md), replacing the old flat MetaTitle/
// MetaDescription pair. Both fields kept side by side during the migration —
// MetaTitle/MetaDescription are not removed yet. Noindex lets an editor
// exclude one entity's detail page from search indexing without touching
// robots logic anywhere else.
type Seo struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Noindex     bool    `json:"noindex,omitempty"`
}

// ProductWithRelations is what read queries (GetAll/GetByID/GetBySlug/
// GetByIDs/GetAdjacent's siblings) return — a Product plus the joined
// category/brand display refs. Create/Update only ever deal with a plain
// *Product, same split as ServiceWithRelations/*Service.
type ProductWithRelations struct {
	*Product
	Category    *CategoryRef
	Brand       *BrandRef
	Tags        []TagRef
	ProductLine *ProductLine
	Options     []ProductOption
	Variants    []*ProductVariant
	// AttributeValues is only ever populated on single-product reads
	// (GetByID/GetBySlug) — see ProductRepository's doc comment.
	AttributeValues []AttributeValueRef
}

// Product is a pure container entity — matching Shopify/Medusa's model, it
// deliberately holds NO sku/mpn/gtin/price/stock of its own. Every sellable
// identity lives exclusively on ProductVariant; a "simple" product (no real
// options) still has exactly one variant, never zero — see
// application.resolveDefaultVariant, which rejects a create/update with an
// empty variant list. Before migration 000007 this entity carried its own
// sku/mpn/gtin/price/stock, used as a silent fallback whenever a product had
// zero variant rows — a "boss variant on Product + optional follower
// variants on ProductVariant" hybrid inconsistent with the rest of the
// variant model (migration 000005). See docs/product-v2-design.md.
type Product struct {
	id               string
	categoryID       string
	brandID          string
	name             string
	slug             string
	description      json.RawMessage
	specs            []SpecItem
	images           []ImageAsset
	labels           []string
	isFeatured       bool
	isPublished      bool
	orderIndex       int
	condition        string
	metaTitle        *string
	metaDescription  *string
	seo              Seo
	productLineID    *string
	shortDescription *string
	warrantyMonths   *int
	warrantyTerms    *string
	// Denormalized read cache of the variant tree — recomputed by the
	// infrastructure layer whenever a variant changes, never mutated
	// through this entity's own Update* methods. defaultVariantID/
	// displayPrice/displayStockStatus/priceMin/priceMax always populated in
	// practice (every product has >=1 variant); variantMpns is the
	// space-joined MPNs of all active variants, feeding search_vector so
	// customers can find a product by the manufacturer code they actually
	// search — not sku, which is internal-only. See docs/product-v2-design.md.
	defaultVariantID   *string
	displayPrice       *int64
	displayStockStatus *string
	priceMin           *int64
	priceMax           *int64
	variantMpns        string
	createdAt          time.Time
	updatedAt          time.Time
	deletedAt          *time.Time
}

// NewProduct validates and creates a new entity from user input.
func NewProduct(
	categoryID, brandID, name, slug string,
	description json.RawMessage,
	specs []SpecItem,
	images []ImageAsset,
	labels []string,
	isFeatured, isPublished bool,
	orderIndex int,
	condition string,
	metaTitle, metaDescription *string,
	seo Seo,
	productLineID, shortDescription *string,
	warrantyMonths *int,
	warrantyTerms *string,
) (*Product, error) {
	fields := map[string][]string{}

	if errs := validateName(name); len(errs) > 0 {
		fields["name"] = errs
	}
	if errs := validateSlug(slug); len(errs) > 0 {
		fields["slug"] = errs
	}
	if errs := validateCategoryID(categoryID); len(errs) > 0 {
		fields["category_id"] = errs
	}
	if errs := validateBrandID(brandID); len(errs) > 0 {
		fields["brand_id"] = errs
	}
	if errs := validateCondition(condition); len(errs) > 0 {
		fields["condition"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	if condition == "" {
		condition = "new"
	}

	now := time.Now()
	return &Product{
		categoryID:       categoryID,
		brandID:          brandID,
		name:             name,
		slug:             slug,
		description:      description,
		specs:            specs,
		images:           images,
		labels:           labels,
		isFeatured:       isFeatured,
		isPublished:      isPublished,
		orderIndex:       orderIndex,
		condition:        condition,
		metaTitle:        metaTitle,
		metaDescription:  metaDescription,
		seo:              seo,
		productLineID:    productLineID,
		shortDescription: shortDescription,
		warrantyMonths:   warrantyMonths,
		warrantyTerms:    warrantyTerms,
		createdAt:        now,
		updatedAt:        now,
	}, nil
}

// RehydrateProduct reconstructs from a trusted DB row — no validation. Only
// the infrastructure layer should call this.
func RehydrateProduct(
	id, categoryID, brandID, name, slug string,
	description json.RawMessage,
	specs []SpecItem,
	images []ImageAsset,
	labels []string,
	isFeatured, isPublished bool,
	orderIndex int,
	condition string,
	metaTitle, metaDescription *string,
	seo Seo,
	productLineID, shortDescription *string,
	warrantyMonths *int,
	warrantyTerms *string,
	defaultVariantID *string,
	displayPrice *int64,
	displayStockStatus *string,
	priceMin, priceMax *int64,
	variantMpns string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Product {
	return &Product{
		id: id, categoryID: categoryID, brandID: brandID,
		name: name, slug: slug,
		description: description, specs: specs,
		images: images, labels: labels,
		isFeatured: isFeatured, isPublished: isPublished, orderIndex: orderIndex,
		condition: condition,
		metaTitle: metaTitle, metaDescription: metaDescription, seo: seo,
		productLineID: productLineID, shortDescription: shortDescription,
		warrantyMonths: warrantyMonths, warrantyTerms: warrantyTerms,
		defaultVariantID: defaultVariantID, displayPrice: displayPrice,
		displayStockStatus: displayStockStatus, priceMin: priceMin, priceMax: priceMax,
		variantMpns: variantMpns,
		createdAt:   createdAt, updatedAt: updatedAt, deletedAt: deletedAt,
	}
}

func (p *Product) ID() string                   { return p.id }
func (p *Product) CategoryID() string           { return p.categoryID }
func (p *Product) BrandID() string              { return p.brandID }
func (p *Product) Name() string                 { return p.name }
func (p *Product) Slug() string                 { return p.slug }
func (p *Product) Description() json.RawMessage { return p.description }
func (p *Product) Specs() []SpecItem            { return p.specs }
func (p *Product) Images() []ImageAsset         { return p.images }
func (p *Product) Labels() []string             { return p.labels }
func (p *Product) IsFeatured() bool             { return p.isFeatured }
func (p *Product) IsPublished() bool            { return p.isPublished }
func (p *Product) OrderIndex() int              { return p.orderIndex }
func (p *Product) Condition() string            { return p.condition }
func (p *Product) MetaTitle() *string           { return p.metaTitle }
func (p *Product) MetaDescription() *string     { return p.metaDescription }
func (p *Product) Seo() Seo                     { return p.seo }
func (p *Product) ProductLineID() *string       { return p.productLineID }
func (p *Product) ShortDescription() *string    { return p.shortDescription }
func (p *Product) WarrantyMonths() *int         { return p.warrantyMonths }
func (p *Product) WarrantyTerms() *string       { return p.warrantyTerms }
func (p *Product) DefaultVariantID() *string    { return p.defaultVariantID }
func (p *Product) DisplayPrice() *int64         { return p.displayPrice }
func (p *Product) DisplayStockStatus() *string  { return p.displayStockStatus }
func (p *Product) PriceMin() *int64             { return p.priceMin }
func (p *Product) PriceMax() *int64             { return p.priceMax }
func (p *Product) VariantMpns() string          { return p.variantMpns }
func (p *Product) CreatedAt() time.Time         { return p.createdAt }
func (p *Product) UpdatedAt() time.Time         { return p.updatedAt }
func (p *Product) DeletedAt() *time.Time        { return p.deletedAt }

func (p *Product) IsDeleted() bool {
	return p.deletedAt != nil
}

func (p *Product) UpdateName(name string) error {
	if errs := validateName(name); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	p.name = name
	p.updatedAt = time.Now()
	return nil
}

func (p *Product) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	p.slug = slug
	p.updatedAt = time.Now()
	return nil
}

func (p *Product) UpdateCategoryID(categoryID string) error {
	if errs := validateCategoryID(categoryID); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"category_id": errs})
	}
	p.categoryID = categoryID
	p.updatedAt = time.Now()
	return nil
}

func (p *Product) UpdateBrandID(brandID string) error {
	if errs := validateBrandID(brandID); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"brand_id": errs})
	}
	p.brandID = brandID
	p.updatedAt = time.Now()
	return nil
}

func (p *Product) UpdateDescription(description json.RawMessage) {
	p.description = description
	p.updatedAt = time.Now()
}

func (p *Product) UpdateSpecs(specs []SpecItem) {
	p.specs = specs
	p.updatedAt = time.Now()
}

func (p *Product) UpdateImages(images []ImageAsset) {
	p.images = images
	p.updatedAt = time.Now()
}

func (p *Product) SetLabels(labels []string) {
	p.labels = labels
	p.updatedAt = time.Now()
}

func (p *Product) SetFeatured(isFeatured bool) {
	p.isFeatured = isFeatured
	p.updatedAt = time.Now()
}

func (p *Product) SetPublished(isPublished bool) {
	p.isPublished = isPublished
	p.updatedAt = time.Now()
}

func (p *Product) Reorder(orderIndex int) {
	p.orderIndex = orderIndex
	p.updatedAt = time.Now()
}

func (p *Product) UpdateCondition(condition string) error {
	if errs := validateCondition(condition); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"condition": errs})
	}
	p.condition = condition
	p.updatedAt = time.Now()
	return nil
}

func (p *Product) UpdateMetaTitle(metaTitle *string) {
	p.metaTitle = metaTitle
	p.updatedAt = time.Now()
}

func (p *Product) UpdateMetaDescription(metaDescription *string) {
	p.metaDescription = metaDescription
	p.updatedAt = time.Now()
}

func (p *Product) UpdateSeo(seo Seo) {
	p.seo = seo
	p.updatedAt = time.Now()
}

func (p *Product) UpdateProductLineID(productLineID *string) {
	p.productLineID = productLineID
	p.updatedAt = time.Now()
}

func (p *Product) UpdateShortDescription(shortDescription *string) {
	p.shortDescription = shortDescription
	p.updatedAt = time.Now()
}

func (p *Product) UpdateWarranty(warrantyMonths *int, warrantyTerms *string) {
	p.warrantyMonths = warrantyMonths
	p.warrantyTerms = warrantyTerms
	p.updatedAt = time.Now()
}

func (p *Product) MarkDeleted(deletedAt time.Time) {
	p.deletedAt = &deletedAt
}

func (p *Product) Restore() {
	p.deletedAt = nil
	p.updatedAt = time.Now()
}

func validateName(name string) []string {
	if name == "" {
		return []string{"name is required"}
	}
	return nil
}

func validateSlug(slug string) []string {
	if slug == "" {
		return []string{"slug is required"}
	}
	return nil
}

// validateCategoryID/validateBrandID exist because products.category_id and
// products.brand_id are both NOT NULL — a deliberate addition beyond what was
// explicitly spelled out for brand/service (whose FK columns are nullable),
// so a bad partial update fails with a clean 400 here instead of a raw
// Postgres NOT NULL constraint violation surfacing as a 500. See
// docs/catalog.md.
func validateCategoryID(categoryID string) []string {
	if categoryID == "" {
		return []string{"category_id is required"}
	}
	return nil
}

func validateBrandID(brandID string) []string {
	if brandID == "" {
		return []string{"brand_id is required"}
	}
	return nil
}

// validateCondition enforces the product_condition Postgres enum's two
// values so an invalid value fails fast with a 400 instead of a raw
// "invalid input value for enum" 500 from Postgres.
func validateCondition(condition string) []string {
	if condition != "" && condition != "new" && condition != "used" {
		return []string{"condition must be 'new' or 'used'"}
	}
	return nil
}

type CreateProductInput struct {
	CategoryID       string
	BrandID          string
	Name             string
	Slug             string
	Description      json.RawMessage
	Specs            []SpecItem
	Images           []ImageAsset
	Labels           []string
	IsFeatured       bool
	IsPublished      bool
	OrderIndex       int
	Condition        string
	MetaTitle        *string
	MetaDescription  *string
	Seo              Seo
	TagIDs           []string
	ProductLineID    *string
	ShortDescription *string
	WarrantyMonths   *int
	WarrantyTerms    *string
	Options          []ProductOptionInput
	// Variants must contain at least one entry — a product with zero
	// variants is rejected by the application layer (resolveDefaultVariant),
	// not silently accepted. See the Product doc comment above.
	Variants        []ProductVariantInput
	AttributeValues []ProductAttributeValueInput
}

// UpdateProductInput is a partial update — nil/unset means "not part of this
// request", same convention as UpdateBrandInput/UpdateServiceInput (a slice
// field left nil is left untouched; sending an explicit empty slice is what
// clears it).
type UpdateProductInput struct {
	ID               string
	CategoryID       *string
	BrandID          *string
	Name             *string
	Slug             *string
	Description      json.RawMessage
	Specs            []SpecItem
	Images           []ImageAsset
	Labels           []string
	IsFeatured       *bool
	IsPublished      *bool
	OrderIndex       *int
	Condition        *string
	MetaTitle        *string
	MetaDescription  *string
	Seo              *Seo
	TagIDs           *[]string
	ProductLineID    *string
	ShortDescription *string
	WarrantyMonths   *int
	WarrantyTerms    *string
	// Options/Variants: nil = leave the whole variant tree untouched,
	// non-nil = replace wholesale — same *[]T "replace if present" convention
	// as TagIDs. Options and Variants are always sent together (a variant's
	// OptionSelections reference the Options in the same request). A
	// non-nil Variants must contain at least one entry — same rule as
	// CreateProductInput.Variants.
	Options  *[]ProductOptionInput
	Variants *[]ProductVariantInput
	// AttributeValues: nil = leave existing product_attribute_values
	// untouched, non-nil = replace wholesale — same convention as Options/
	// Variants.
	AttributeValues *[]ProductAttributeValueInput
}

// ProductFilter is bare list scoping (pagination + basic ID/flag matching) —
// deliberately no search/price-range/spec-facet/sort fields; those belonged
// to the removed facet/search system (see docs/catalog.md history) and will
// return, if at all, as part of the upcoming attribute-set redesign.
type ProductFilter struct {
	CategoryID     *string
	CategoryIDs    []string
	BrandID        *string
	BrandIDs       []string
	ProductLineID  *string
	IsFeatured     *bool
	IsPublished    *bool
	Limit          int
	Offset         int
	IncludeDeleted bool
}
