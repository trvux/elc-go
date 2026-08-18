package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Variant stock status — replaces products.stock_status's free-text label
// with a real enum reflecting how this business actually fulfills orders
// (order from đại lý on demand, not multi-warehouse stock) — see
// docs/product-v2-design.md "Explicitly out of scope".
const (
	StockStatusInStock           = "in_stock"
	StockStatusOrderFromSupplier = "order_from_supplier"
	StockStatusDiscontinued      = "discontinued"
)

// ProductLine models "dòng sản phẩm" — e.g. Daikin's FTF/FTKB/FTKF/FTKY/FTKZ
// tiers. Lives inside the product module (not its own bounded context)
// since it currently has exactly one consumer, Product.
type ProductLine struct {
	id          string
	brandID     string
	categoryID  *string
	code        string
	name        string
	tierRank    int
	description *string
	// mpnPrefixes are the manufacturer MPN prefixes that identify a product
	// as belonging to this line (e.g. Daikin's "Dòng Tiêu Chuẩn" line has
	// prefixes ["FTF","FTC"]) — a real list, not a single delimited string,
	// so the admin UI can show/edit each prefix as its own chip and future
	// backfills can match against it directly instead of a `code` string
	// like "FTF-FTC". `code` itself stays a short, single, human-chosen
	// identifier for the line (unique per brand), unrelated to how many
	// prefixes it covers.
	mpnPrefixes []string
	createdAt   time.Time
	updatedAt   time.Time
	deletedAt   *time.Time
}

func NewProductLine(brandID, categoryID *string, code, name string, tierRank int, description *string, mpnPrefixes []string) (*ProductLine, error) {
	fields := map[string][]string{}
	if brandID == nil || *brandID == "" {
		fields["brand_id"] = []string{"brand_id is required"}
	}
	if code == "" {
		fields["code"] = []string{"code is required"}
	}
	if name == "" {
		fields["name"] = []string{"name is required"}
	}
	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}
	now := time.Now()
	return &ProductLine{
		brandID: *brandID, categoryID: categoryID, code: code, name: name,
		tierRank: tierRank, description: description, mpnPrefixes: mpnPrefixes,
		createdAt: now, updatedAt: now,
	}, nil
}

func RehydrateProductLine(id, brandID string, categoryID *string, code, name string, tierRank int, description *string, mpnPrefixes []string, createdAt, updatedAt time.Time, deletedAt *time.Time) *ProductLine {
	return &ProductLine{
		id: id, brandID: brandID, categoryID: categoryID, code: code, name: name,
		tierRank: tierRank, description: description, mpnPrefixes: mpnPrefixes,
		createdAt: createdAt, updatedAt: updatedAt, deletedAt: deletedAt,
	}
}

func (l *ProductLine) ID() string            { return l.id }
func (l *ProductLine) BrandID() string       { return l.brandID }
func (l *ProductLine) CategoryID() *string   { return l.categoryID }
func (l *ProductLine) Code() string          { return l.code }
func (l *ProductLine) Name() string          { return l.name }
func (l *ProductLine) TierRank() int         { return l.tierRank }
func (l *ProductLine) Description() *string  { return l.description }
func (l *ProductLine) MpnPrefixes() []string { return l.mpnPrefixes }
func (l *ProductLine) CreatedAt() time.Time  { return l.createdAt }
func (l *ProductLine) UpdatedAt() time.Time  { return l.updatedAt }
func (l *ProductLine) DeletedAt() *time.Time { return l.deletedAt }
func (l *ProductLine) IsDeleted() bool       { return l.deletedAt != nil }

func (l *ProductLine) MarkDeleted(deletedAt time.Time) { l.deletedAt = &deletedAt }
func (l *ProductLine) Restore()                        { l.deletedAt = nil; l.updatedAt = time.Now() }

func (l *ProductLine) Update(categoryID *string, name string, tierRank int, description *string, mpnPrefixes []string) error {
	if name == "" {
		return apperr.NewValidationError("validation failed", map[string][]string{"name": {"name is required"}})
	}
	l.categoryID = categoryID
	l.name = name
	l.tierRank = tierRank
	l.description = description
	l.mpnPrefixes = mpnPrefixes
	l.updatedAt = time.Now()
	return nil
}

type CreateProductLineInput struct {
	BrandID     string
	CategoryID  *string
	Code        string
	Name        string
	TierRank    int
	Description *string
	MpnPrefixes []string
}

type UpdateProductLineInput struct {
	ID          string
	CategoryID  *string
	Name        string
	TierRank    int
	Description *string
	MpnPrefixes []string
}

// ProductOptionValue is always loaded/written as part of its parent
// ProductOption — it has no independent lifecycle of its own.
type ProductOptionValue struct {
	ID         string
	Value      string
	OrderIndex int
}

// ProductOption + its values. Options are always replaced wholesale on
// product update (delete-then-reinsert), same convention this project
// already uses for project_type_category — there is no independent history
// worth preserving for "Màu sắc: Đỏ" the way there is for a variant's own
// identity (SKU/cost/eventual order references), see
// docs/product-v2-design.md's rollout notes.
type ProductOption struct {
	ID         string
	ProductID  string
	Name       string
	OrderIndex int
	Values     []ProductOptionValue
}

// VariantRef is a lightweight read-only reference to a sibling variant —
// used by ProductVariantComponent to show a bundle part's MPN/SKU without a
// full nested ProductVariant, same reasoning as CategoryRef/BrandRef.
type VariantRef struct {
	ID  string
	MPN string
	SKU string
}

type ProductVariantComponent struct {
	ID        string
	Component VariantRef
	Quantity  int
	Role      *string
}

// ProductVariant is the real purchasable unit — one MPN, one price, one
// stock status. mpn is required (this is what customers actually search
// for); sku is this business's own internal warehouse code, never rendered
// to the public. isStandalone=false marks a variant that only exists as a
// bundle component (e.g. "Dàn lạnh FTKB25") and should be hidden from a
// variant picker.
type ProductVariant struct {
	id              string
	productID       string
	mpn             string
	sku             string
	gtin            *string
	isDefault       bool
	isStandalone    bool
	stockStatus     string
	leadTimeDays    *int
	costPrice       *int64
	originalPrice   int64
	salePrice       *int64
	discountPercent float64
	weight          *float64
	isActive        bool
	orderIndex      int
	optionValueIDs  []string
	components      []ProductVariantComponent
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

func RehydrateProductVariant(
	id, productID, mpn, sku string,
	gtin *string,
	isDefault, isStandalone bool,
	stockStatus string,
	leadTimeDays *int,
	costPrice *int64,
	originalPrice int64,
	salePrice *int64,
	discountPercent float64,
	weight *float64,
	isActive bool,
	orderIndex int,
	optionValueIDs []string,
	components []ProductVariantComponent,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *ProductVariant {
	return &ProductVariant{
		id: id, productID: productID, mpn: mpn, sku: sku, gtin: gtin,
		isDefault: isDefault, isStandalone: isStandalone, stockStatus: stockStatus,
		leadTimeDays: leadTimeDays, costPrice: costPrice,
		originalPrice: originalPrice, salePrice: salePrice, discountPercent: discountPercent,
		weight: weight, isActive: isActive, orderIndex: orderIndex,
		optionValueIDs: optionValueIDs, components: components,
		createdAt: createdAt, updatedAt: updatedAt, deletedAt: deletedAt,
	}
}

func (v *ProductVariant) ID() string                            { return v.id }
func (v *ProductVariant) ProductID() string                     { return v.productID }
func (v *ProductVariant) MPN() string                           { return v.mpn }
func (v *ProductVariant) SKU() string                           { return v.sku }
func (v *ProductVariant) GTIN() *string                         { return v.gtin }
func (v *ProductVariant) IsDefault() bool                       { return v.isDefault }
func (v *ProductVariant) IsStandalone() bool                    { return v.isStandalone }
func (v *ProductVariant) StockStatus() string                   { return v.stockStatus }
func (v *ProductVariant) LeadTimeDays() *int                    { return v.leadTimeDays }
func (v *ProductVariant) CostPrice() *int64                     { return v.costPrice }
func (v *ProductVariant) OriginalPrice() int64                  { return v.originalPrice }
func (v *ProductVariant) SalePrice() *int64                     { return v.salePrice }
func (v *ProductVariant) DiscountPercent() float64              { return v.discountPercent }
func (v *ProductVariant) Weight() *float64                      { return v.weight }
func (v *ProductVariant) IsActive() bool                        { return v.isActive }
func (v *ProductVariant) OrderIndex() int                       { return v.orderIndex }
func (v *ProductVariant) OptionValueIDs() []string              { return v.optionValueIDs }
func (v *ProductVariant) Components() []ProductVariantComponent { return v.components }
func (v *ProductVariant) CreatedAt() time.Time                  { return v.createdAt }
func (v *ProductVariant) UpdatedAt() time.Time                  { return v.updatedAt }
func (v *ProductVariant) DeletedAt() *time.Time                 { return v.deletedAt }
func (v *ProductVariant) IsDeleted() bool                       { return v.deletedAt != nil }

// DisplayPrice mirrors NormalizeProductPrice's "sale price wins if set"
// convention at the variant level.
// DisplayPrice treats a stored salePrice of exactly 0 the same as "not
// set" — legacy data migrated from the pre-Go schema stores 0 (not NULL)
// to mean "no sale price", same convention NormalizeProductPrice already
// applies to the old flat product-level fields (see price.go's Case 3).
// Rows created going forward through this entity should never persist a
// literal 0 salePrice in the first place, but this guards existing data.
func (v *ProductVariant) DisplayPrice() int64 {
	if v.salePrice != nil && *v.salePrice > 0 {
		return *v.salePrice
	}
	return v.originalPrice
}

// ProductOptionInput/ProductVariantInput/VariantOptionSelection/
// VariantComponentInput are transient create/update DTOs (application layer
// input), distinct from the persisted ProductOption/ProductVariant/
// ProductVariantComponent entities above.
type ProductOptionInput struct {
	Name   string
	Values []string
}

// VariantOptionSelection identifies which option-value a variant represents
// by (option name, value) rather than by ID — IDs don't exist yet for
// options/values created in the same request, and the repository resolves
// this to real IDs after inserting product_options/product_option_values
// (see infrastructure/postgres_repository.go), staying consistent with this
// project's existing "DB generates IDs, Go never does" convention rather
// than introducing client-generated UUIDs just for this.
type VariantOptionSelection struct {
	OptionName string
	Value      string
}

// VariantComponentInput references a sibling variant by its index into the
// same request's Variants slice (not by ID — the component variant may not
// have an ID yet either, if it's being created in the same call).
type VariantComponentInput struct {
	ComponentIndex int
	Quantity       int
	Role           *string
}

type ProductVariantInput struct {
	MPN              string
	SKU              *string
	GTIN             *string
	IsDefault        bool
	IsComponentOnly  bool // inverse of is_standalone; false (zero value) is the correct default
	StockStatus      string
	LeadTimeDays     *int
	CostPrice        *int64
	OriginalPrice    int64
	SalePrice        *int64
	DiscountPercent  float64
	Weight           *float64
	IsActive         bool
	OrderIndex       int
	OptionSelections []VariantOptionSelection
	Components       []VariantComponentInput
}
