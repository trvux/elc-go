package domain

import "context"

// ProductListResult is a pure repository-result shape (not a persistent
// entity), same spirit as ServiceWithRelations.
type ProductListResult struct {
	Products   []*ProductWithRelations
	TotalCount int
}

// ProductRepository — GetByID/GetBySlug/GetByIDs return *ProductWithRelations
// (not the bare *Product the original task sketch showed) so the HTTP
// response can carry nested category/brand objects without a second round
// trip; Create/Update return a plain *Product, exactly the same split
// service already uses (ServiceWithRelations for reads, *Service for
// writes) — see internal/service/domain/repository.go for the precedent
// this follows.
type ProductRepository interface {
	GetAll(ctx context.Context, filter ProductFilter) (*ProductListResult, error)
	Count(ctx context.Context, filter ProductFilter) (int, error)
	GetByID(ctx context.Context, id string) (*ProductWithRelations, error)
	GetBySlug(ctx context.Context, slug string) (*ProductWithRelations, error)
	GetByIDs(ctx context.Context, ids []string) ([]*ProductWithRelations, error)
	// GetByIDsWithAttributeValues is GetByIDs plus each product's
	// AttributeValues — used by the Comparison feature, which is the first
	// caller that needs specs across more than one product at a time.
	GetByIDsWithAttributeValues(ctx context.Context, ids []string) ([]*ProductWithRelations, error)
	// tagIDs on Create is the initial tag set; on Update, nil means "leave
	// tags untouched", a non-nil pointer means "replace all tags with this
	// set" — same convention as project's Categories/ServiceIDs. options/
	// variants follow the same nil-vs-non-nil convention on Update (nil =
	// leave the variant tree untouched); on Create they're the initial tree.
	// attributeValues follows the same nil-vs-non-nil convention as options/
	// variants on Update: nil leaves existing product_attribute_values
	// untouched, a non-nil pointer replaces the whole set. Unlike options/
	// variants, attribute values are only ever fetched on single-product
	// reads (GetByID/GetBySlug) — see attachAttributeValuesToProducts's doc
	// comment — never on GetAll/GetByIDs, since specs aren't needed for list/
	// card rendering.
	Create(ctx context.Context, product *Product, tagIDs []string, options []ProductOptionInput, variants []ProductVariantInput, attributeValues []ProductAttributeValueInput) (*Product, error)
	Update(ctx context.Context, product *Product, tagIDs *[]string, options *[]ProductOptionInput, variants *[]ProductVariantInput, attributeValues *[]ProductAttributeValueInput) (*Product, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}

// CatalogPageRepository manages the single product_catalog_page row —
// deliberately no Create/Delete/List, the row already exists (seeded by
// migration) and is never removed.
type CatalogPageRepository interface {
	Get(ctx context.Context) (*CatalogPage, error)
	Update(ctx context.Context, input UpdateCatalogPageInput) (*CatalogPage, error)
}

// ProductLineRepository is intentionally a separate interface (not folded
// into ProductRepository) even though it lives in the same Go package/table
// migration — it's a simple independent CRUD lookup, same shape as
// brand/category's own repositories, not something Product's read/write
// paths depend on beyond a plain ID reference.
type ProductLineRepository interface {
	List(ctx context.Context, brandID *string, includeDeleted bool) ([]*ProductLine, error)
	GetByID(ctx context.Context, id string) (*ProductLine, error)
	Create(ctx context.Context, line *ProductLine) (*ProductLine, error)
	Update(ctx context.Context, line *ProductLine) (*ProductLine, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
