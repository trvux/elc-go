package domain

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/platform/seo"
)

// ImageAsset re-exports the shared media type so callers outside this
// package can write domain.ImageAsset without also importing
// internal/platform/media directly.
type ImageAsset = media.ImageAsset

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
	// CapacitySiblings is only ever populated on single-product reads
	// (GetByID/GetBySlug) — other published products that are the same
	// model at a different HP/capacity (see attachCapacitySiblings), never
	// on GetByIDs/list reads. Nil when there are no siblings (a
	// single-capacity model), same "don't attach a length-1 selector"
	// convention as the frontend's variant option switcher.
	CapacitySiblings []CapacitySibling
}

// ProductStatus is the publish lifecycle: employees submit a draft for
// review, an owner/admin approves it to published or rejects it back to
// draft (with a reason), and a published product can later be archived
// (discontinued but its URL kept alive for existing backlinks/SEO rather
// than 404ing). There is no separate "rejected" terminal state — a
// rejection's only actionable next step is revise-and-resubmit, so it folds
// back into draft plus RejectionReason.
type ProductStatus string

const (
	ProductStatusDraft     ProductStatus = "draft"
	ProductStatusProposed  ProductStatus = "proposed"
	ProductStatusPublished ProductStatus = "published"
	ProductStatusArchived  ProductStatus = "archived"
)

func (s ProductStatus) IsValid() bool {
	switch s {
	case ProductStatusDraft, ProductStatusProposed, ProductStatusPublished, ProductStatusArchived:
		return true
	default:
		return false
	}
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
	id              string
	categoryID      string
	brandID         string
	name            string
	slug            string
	description     json.RawMessage
	images          []ImageAsset
	isFeatured      bool
	status          ProductStatus
	rejectionReason *string
	orderIndex      int
	metaTitle       *string
	metaDescription *string
	productLineID   *string
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
// NewProduct always starts a product as ProductStatusDraft — status only
// ever advances through the explicit transition methods below
// (SubmitForReview/Approve/Reject/Archive/Unarchive), never as a
// caller-supplied value, so the approval workflow can't be bypassed by
// passing status straight into create/update.
func NewProduct(
	categoryID, brandID, name, slug string,
	description json.RawMessage,
	images []ImageAsset,
	isFeatured bool,
	orderIndex int,
	metaTitle, metaDescription *string,
	productLineID *string,
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
	if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
		fields["metaTitle"] = errs
	}
	if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
		fields["metaDescription"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Product{
		categoryID:      categoryID,
		brandID:         brandID,
		name:            name,
		slug:            slug,
		description:     description,
		images:          images,
		isFeatured:      isFeatured,
		status:          ProductStatusDraft,
		orderIndex:      orderIndex,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		productLineID:   productLineID,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateProduct reconstructs from a trusted DB row — no validation. Only
// the infrastructure layer should call this.
func RehydrateProduct(
	id, categoryID, brandID, name, slug string,
	description json.RawMessage,
	images []ImageAsset,
	isFeatured bool,
	status ProductStatus,
	rejectionReason *string,
	orderIndex int,
	metaTitle, metaDescription *string,
	productLineID *string,
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
		description: description,
		images:      images,
		isFeatured:  isFeatured, status: status, rejectionReason: rejectionReason,
		orderIndex: orderIndex,
		metaTitle:  metaTitle, metaDescription: metaDescription,
		productLineID:    productLineID,
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
func (p *Product) Images() []ImageAsset         { return p.images }
func (p *Product) IsFeatured() bool             { return p.isFeatured }
func (p *Product) Status() ProductStatus        { return p.status }
func (p *Product) RejectionReason() *string     { return p.rejectionReason }
func (p *Product) OrderIndex() int              { return p.orderIndex }
func (p *Product) MetaTitle() *string           { return p.metaTitle }
func (p *Product) MetaDescription() *string     { return p.metaDescription }
func (p *Product) ProductLineID() *string       { return p.productLineID }
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

func (p *Product) UpdateImages(images []ImageAsset) {
	p.images = images
	p.updatedAt = time.Now()
}

func (p *Product) SetFeatured(isFeatured bool) {
	p.isFeatured = isFeatured
	p.updatedAt = time.Now()
}

func (p *Product) Reorder(orderIndex int) {
	p.orderIndex = orderIndex
	p.updatedAt = time.Now()
}

// SubmitForReview moves a draft (freshly created, or sent back by a
// rejection) into proposed, awaiting an owner/admin's Approve or Reject.
func (p *Product) SubmitForReview() error {
	if p.status != ProductStatusDraft {
		return apperr.NewValidationError("validation failed", map[string][]string{
			"status": {"only a draft product can be submitted for review"},
		})
	}
	p.status = ProductStatusProposed
	p.updatedAt = time.Now()
	return nil
}

// Approve publishes a proposed product, clearing any earlier rejection note.
func (p *Product) Approve() error {
	if p.status != ProductStatusProposed {
		return apperr.NewValidationError("validation failed", map[string][]string{
			"status": {"only a proposed product can be approved"},
		})
	}
	p.status = ProductStatusPublished
	p.rejectionReason = nil
	p.updatedAt = time.Now()
	return nil
}

// Reject sends a proposed product back to draft with a reason so the
// submitting employee knows what to fix — there is no separate "rejected"
// state, the only actionable next step is revise-and-resubmit.
func (p *Product) Reject(reason string) error {
	if p.status != ProductStatusProposed {
		return apperr.NewValidationError("validation failed", map[string][]string{
			"status": {"only a proposed product can be rejected"},
		})
	}
	p.status = ProductStatusDraft
	p.rejectionReason = &reason
	p.updatedAt = time.Now()
	return nil
}

// Archive discontinues a published product while keeping its URL/row alive
// (existing backlinks/SEO), rather than deleting it.
func (p *Product) Archive() error {
	if p.status != ProductStatusPublished {
		return apperr.NewValidationError("validation failed", map[string][]string{
			"status": {"only a published product can be archived"},
		})
	}
	p.status = ProductStatusArchived
	p.updatedAt = time.Now()
	return nil
}

// Unarchive re-lists a discontinued product directly back to published —
// it was already approved once, so it doesn't need to go through review
// again.
func (p *Product) Unarchive() error {
	if p.status != ProductStatusArchived {
		return apperr.NewValidationError("validation failed", map[string][]string{
			"status": {"only an archived product can be unarchived"},
		})
	}
	p.status = ProductStatusPublished
	p.updatedAt = time.Now()
	return nil
}

func (p *Product) UpdateMetaTitle(metaTitle *string) error {
	if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaTitle": errs})
	}
	p.metaTitle = metaTitle
	p.updatedAt = time.Now()
	return nil
}

func (p *Product) UpdateMetaDescription(metaDescription *string) error {
	if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaDescription": errs})
	}
	p.metaDescription = metaDescription
	p.updatedAt = time.Now()
	return nil
}

func (p *Product) UpdateProductLineID(productLineID *string) {
	p.productLineID = productLineID
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

// CreateProductInput deliberately carries no status field — NewProduct
// always starts a product as ProductStatusDraft, and status only ever
// advances through the dedicated submit/approve/reject/archive endpoints
// (see ProductStatus doc comment), never through create/update payloads.
type CreateProductInput struct {
	CategoryID      string
	BrandID         string
	Name            string
	Slug            string
	Description     json.RawMessage
	Images          []ImageAsset
	IsFeatured      bool
	OrderIndex      int
	MetaTitle       *string
	MetaDescription *string
	TagIDs          []string
	ProductLineID   *string
	Options         []ProductOptionInput
	// Variants must contain at least one entry — a product with zero
	// variants is rejected by the application layer (resolveDefaultVariant),
	// not silently accepted. See the Product doc comment above.
	Variants        []ProductVariantInput
	AttributeValues []ProductAttributeValueInput
}

// UpdateProductInput is a partial update — nil/unset means "not part of this
// request", same convention as UpdateBrandInput/UpdateServiceInput (a slice
// field left nil is left untouched; sending an explicit empty slice is what
// clears it). Deliberately no status field — see CreateProductInput's doc
// comment; status only moves through submit/approve/reject/archive.
type UpdateProductInput struct {
	ID              string
	CategoryID      *string
	BrandID         *string
	Name            *string
	Slug            *string
	Description     json.RawMessage
	Images          []ImageAsset
	IsFeatured      *bool
	OrderIndex      *int
	MetaTitle       *string
	MetaDescription *string
	TagIDs          *[]string
	ProductLineID   *string
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

// Sort values accepted by ProductFilter.SortBy — empty means the default
// (is_featured DESC, order_index ASC, unchanged from before this filter set
// existed).
const (
	SortByPriceAsc  = "price_asc"
	SortByPriceDesc = "price_desc"
	SortByNewest    = "newest"
)

// ProductFilter is list scoping (pagination + basic ID/flag matching) plus
// search/price-range/attribute-facet dimensions, rebuilt on top of the
// structured attribute system (attribute_definitions/product_attribute_values)
// after the original free-text-spec-based facet/search system was removed.
type ProductFilter struct {
	CategoryID     *string
	CategoryIDs    []string
	BrandID        *string
	BrandIDs       []string
	ProductLineID  *string
	IsFeatured     *bool
	Status         *ProductStatus
	Limit          int
	Offset         int
	IncludeDeleted bool

	// Search matches against product name + variant MPNs (search_vector),
	// with a trigram similarity fallback for typos/partial matches.
	Search string
	// MinPrice/MaxPrice filter on the denormalized display_price column.
	MinPrice *int64
	MaxPrice *int64
	// SortBy is one of the SortBy* constants above, or "" for the default.
	SortBy string
	// AttributeTokens is a flat list of "code:value" tokens (e.g.
	// "cong_suat_hp:1.5") — grouped by code at query time: OR within the
	// same code, AND across different codes. Matches facet_tokens.
	AttributeTokens []string
	// AttributeRanges filters number-type attributes by [min, max] (either
	// bound may be nil), keyed by attribute code. Queried directly against
	// product_attribute_values.value_number — number attributes aren't
	// discrete/token-facetable.
	AttributeRanges map[string][2]*float64
}

// ProductFacets accompanies a ProductListResult — aggregate counts/ranges
// computed over the filtered set, each dimension excluding its own filter
// (so e.g. switching brand stays visible as an option while one brand is
// currently selected — standard faceted-search "exclude own dimension"
// technique).
type ProductFacets struct {
	Brands     []BrandFacet
	Categories []CategoryFacet // only populated when the page scope spans >1 category (brand/group pages)
	Price      PriceFacet
	Attributes []AttributeFacet
}

type BrandFacet struct {
	ID, Name, Slug, LogoURL string
	Count                   int
}

type CategoryFacet struct {
	ID, Name, Slug string
	Count          int
}

type PriceFacet struct {
	Min, Max int64
	// Buckets are ready-to-click suggested ranges (exact distinct prices if
	// few enough, otherwise round-number ranges) — so the FE can offer
	// preset buttons instead of asking the shopper to type a number they
	// have no reference point for. See BuildNumberBuckets.
	Buckets []NumberBucket
}

// AttributeFacet is a filter control for one AttributeDefinition, scoped to
// whichever categories are actually present in the current page — Options
// is populated for select/multiselect/boolean (discrete, token-facetable);
// Min/Max/Buckets is populated for number (range-facetable) instead.
type AttributeFacet struct {
	Code, Name, DataType string
	GroupLabel           *string
	Unit                 *string
	Options              []AttributeFacetOption
	Min, Max             *float64
	Buckets              []NumberBucket
}

type AttributeFacetOption struct {
	Value string
	Count int
}

// NumberBucket is one suggested range for a number-type facet (price or a
// number attribute). Min == Max means an exact value (used when the
// underlying data only has a handful of distinct values, e.g. a fixed set
// of gas-pipe lengths) rather than a genuine range.
type NumberBucket struct {
	Min, Max float64
	Count    int
}
