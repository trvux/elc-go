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
// module's own SQL (internal/catalog/infrastructure) rather than importing
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
	Category *CategoryRef
	Brand    *BrandRef
	Tags     []TagRef
}

type Product struct {
	id              string
	categoryID      string
	brandID         string
	name            string
	sku             string
	slug            string
	description     json.RawMessage
	specs           []SpecItem
	normalizedSpecs []string
	images          []ImageAsset
	labels          []string
	originalPrice   int64
	salePrice       *int64
	discountPercent float64
	isFeatured      bool
	isPublished     bool
	orderIndex      int
	stockStatus     string
	condition       string
	metaTitle       *string
	metaDescription *string
	seo             Seo
	mpn             *string
	gtin            *string
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// NewProduct validates and creates a new entity from user input.
// normalizedSpecs is a derived value — the application layer must compute it
// via NormalizeProductSpecs(name, specs) before calling this (or before
// calling UpdateSpecs), never inside the domain constructor itself; see
// spec_normalizer.go and application/create_product.go.
func NewProduct(
	categoryID, brandID, name, sku, slug string,
	description json.RawMessage,
	specs []SpecItem,
	normalizedSpecs []string,
	images []ImageAsset,
	labels []string,
	originalPrice int64,
	salePrice *int64,
	discountPercent float64,
	isFeatured, isPublished bool,
	orderIndex int,
	stockStatus, condition string,
	metaTitle, metaDescription, mpn, gtin *string,
	seo Seo,
) (*Product, error) {
	fields := map[string][]string{}

	if errs := validateName(name); len(errs) > 0 {
		fields["name"] = errs
	}
	if errs := validateSlug(slug); len(errs) > 0 {
		fields["slug"] = errs
	}
	if errs := validateSKU(sku); len(errs) > 0 {
		fields["sku"] = errs
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

	if stockStatus == "" {
		stockStatus = "in_stock"
	}
	if condition == "" {
		condition = "new"
	}

	now := time.Now()
	return &Product{
		categoryID:      categoryID,
		brandID:         brandID,
		name:            name,
		sku:             sku,
		slug:            slug,
		description:     description,
		specs:           specs,
		normalizedSpecs: normalizedSpecs,
		images:          images,
		labels:          labels,
		originalPrice:   originalPrice,
		salePrice:       salePrice,
		discountPercent: discountPercent,
		isFeatured:      isFeatured,
		isPublished:     isPublished,
		orderIndex:      orderIndex,
		stockStatus:     stockStatus,
		condition:       condition,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		seo:             seo,
		mpn:             mpn,
		gtin:            gtin,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateProduct reconstructs from a trusted DB row — no validation. Only
// the infrastructure layer should call this.
func RehydrateProduct(
	id, categoryID, brandID, name, sku, slug string,
	description json.RawMessage,
	specs []SpecItem,
	normalizedSpecs []string,
	images []ImageAsset,
	labels []string,
	originalPrice int64,
	salePrice *int64,
	discountPercent float64,
	isFeatured, isPublished bool,
	orderIndex int,
	stockStatus, condition string,
	metaTitle, metaDescription, mpn, gtin *string,
	seo Seo,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Product {
	return &Product{
		id: id, categoryID: categoryID, brandID: brandID,
		name: name, sku: sku, slug: slug,
		description: description, specs: specs, normalizedSpecs: normalizedSpecs,
		images: images, labels: labels,
		originalPrice: originalPrice, salePrice: salePrice, discountPercent: discountPercent,
		isFeatured: isFeatured, isPublished: isPublished, orderIndex: orderIndex,
		stockStatus: stockStatus, condition: condition,
		metaTitle: metaTitle, metaDescription: metaDescription, seo: seo, mpn: mpn, gtin: gtin,
		createdAt: createdAt, updatedAt: updatedAt, deletedAt: deletedAt,
	}
}

func (p *Product) ID() string                   { return p.id }
func (p *Product) CategoryID() string           { return p.categoryID }
func (p *Product) BrandID() string              { return p.brandID }
func (p *Product) Name() string                 { return p.name }
func (p *Product) SKU() string                  { return p.sku }
func (p *Product) Slug() string                 { return p.slug }
func (p *Product) Description() json.RawMessage { return p.description }
func (p *Product) Specs() []SpecItem            { return p.specs }
func (p *Product) NormalizedSpecs() []string    { return p.normalizedSpecs }
func (p *Product) Images() []ImageAsset         { return p.images }
func (p *Product) Labels() []string             { return p.labels }
func (p *Product) OriginalPrice() int64         { return p.originalPrice }
func (p *Product) SalePrice() *int64            { return p.salePrice }
func (p *Product) DiscountPercent() float64     { return p.discountPercent }
func (p *Product) IsFeatured() bool             { return p.isFeatured }
func (p *Product) IsPublished() bool            { return p.isPublished }
func (p *Product) OrderIndex() int              { return p.orderIndex }
func (p *Product) StockStatus() string          { return p.stockStatus }
func (p *Product) Condition() string            { return p.condition }
func (p *Product) MetaTitle() *string           { return p.metaTitle }
func (p *Product) MetaDescription() *string     { return p.metaDescription }
func (p *Product) Seo() Seo                     { return p.seo }
func (p *Product) MPN() *string                 { return p.mpn }
func (p *Product) GTIN() *string                { return p.gtin }
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

func (p *Product) UpdateSKU(sku string) error {
	if errs := validateSKU(sku); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"sku": errs})
	}
	p.sku = sku
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

// UpdateSpecs takes specs and normalizedSpecs together on purpose — the two
// must never drift out of sync (normalizedSpecs is entirely derived from
// specs plus the product name, see NormalizeProductSpecs). The application
// layer is responsible for recomputing normalizedSpecs from the *new* specs
// (and current/new name) before calling this, exactly like service's
// UpdatePricing takes originalPrice/discountPercent together — see
// application/update_product.go.
func (p *Product) UpdateSpecs(specs []SpecItem, normalizedSpecs []string) {
	p.specs = specs
	p.normalizedSpecs = normalizedSpecs
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

// UpdatePricing takes all three price fields together — the application
// layer must resolve them via NormalizeProductPrice first (see
// application/create_product.go, application/update_product.go), so the
// three fields are always a consistent, reconciled triple. This mirrors the
// old TS behavior of normalizeProductPrice, but applied at write-time instead
// of read-time — see docs/catalog.md for why.
func (p *Product) UpdatePricing(originalPrice int64, salePrice *int64, discountPercent float64) {
	p.originalPrice = originalPrice
	p.salePrice = salePrice
	p.discountPercent = discountPercent
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

func (p *Product) UpdateStockStatus(stockStatus string) {
	p.stockStatus = stockStatus
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

func (p *Product) UpdateMPN(mpn *string) {
	p.mpn = mpn
	p.updatedAt = time.Now()
}

func (p *Product) UpdateGTIN(gtin *string) {
	p.gtin = gtin
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

func validateSKU(sku string) []string {
	if sku == "" {
		return []string{"sku is required"}
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
	CategoryID      string
	BrandID         string
	Name            string
	SKU             string
	Slug            string
	Description     json.RawMessage
	Specs           []SpecItem
	Images          []ImageAsset
	Labels          []string
	OriginalPrice   int64
	SalePrice       *int64
	DiscountPercent float64
	IsFeatured      bool
	IsPublished     bool
	OrderIndex      int
	StockStatus     string
	Condition       string
	MetaTitle       *string
	MetaDescription *string
	Seo             Seo
	MPN             *string
	GTIN            *string
	TagIDs          []string
}

// UpdateProductInput is a partial update — nil/unset means "not part of this
// request", same convention as UpdateBrandInput/UpdateServiceInput (a slice
// field left nil is left untouched; sending an explicit empty slice is what
// clears it). OriginalPrice/SalePrice/DiscountPercent are handled together in
// the application layer, mirroring UpdateServiceInput's OriginalPrice/
// DiscountPercent pairing, but widened to a triple here since catalog
// reconciles all three through NormalizeProductPrice — see
// application/update_product.go.
type UpdateProductInput struct {
	ID              string
	CategoryID      *string
	BrandID         *string
	Name            *string
	SKU             *string
	Slug            *string
	Description     json.RawMessage
	Specs           []SpecItem
	Images          []ImageAsset
	Labels          []string
	OriginalPrice   *int64
	SalePrice       *int64
	DiscountPercent *float64
	IsFeatured      *bool
	IsPublished     *bool
	OrderIndex      *int
	StockStatus     *string
	Condition       *string
	MetaTitle       *string
	MetaDescription *string
	Seo             *Seo
	MPN             *string
	GTIN            *string
	TagIDs          *[]string
}

type ProductSortBy string

const (
	SortByPriceAsc     ProductSortBy = "price_asc"
	SortByPriceDesc    ProductSortBy = "price_desc"
	SortByNewest       ProductSortBy = "newest"
	SortByPopularity   ProductSortBy = "popularity"
	SortByDiscountDesc ProductSortBy = "discount_desc"
)

type ProductFilter struct {
	CategoryID     *string
	CategoryIDs    []string
	BrandID        *string
	BrandIDs       []string
	BrandSlugs     []string
	IsFeatured     *bool
	IsPublished    *bool
	Search         string
	MinPrice       *int64
	MaxPrice       *int64
	SortBy         string
	Condition      string
	Specs          map[string][]string
	Limit          int
	Offset         int
	IncludeDeleted bool
}
