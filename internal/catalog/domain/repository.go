package domain

import "context"

// BrandFacet/SpecFacet/ProductFacets/ProductListResult/AdjacentProduct are
// pure repository-result shapes (not persistent entities), same spirit as
// ServiceWithRelations but for the aggregate list-with-facets read that
// catalog needs and service doesn't.
type BrandFacet struct {
	ID   string
	Name string
	Slug string
}

// SpecFacet groups the flat "Label::Value" normalized_specs entries back
// into one UI-facing facet per label, e.g. {Label: "Công suất", Values:
// ["1 HP", "1.5 HP", "2 HP"]}.
type SpecFacet struct {
	Label  string
	Values []string
}

type ProductFacets struct {
	Brands   []BrandFacet
	Specs    []SpecFacet
	MinPrice int64
	MaxPrice int64
}

type ProductListResult struct {
	Products   []*ProductWithRelations
	TotalCount int
	Facets     ProductFacets
}

type AdjacentProduct struct {
	Name string
	Slug string
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
	Create(ctx context.Context, product *Product) (*Product, error)
	Update(ctx context.Context, product *Product) (*Product, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	// GetAdjacent resolves prev/next within categoryID first, falling back to
	// the full published catalog when that category has fewer than 2
	// published siblings — see application/get_adjacent_products.go for why
	// the fallback *decision* still lives in the application layer even
	// though both queries run here.
	GetAdjacent(ctx context.Context, categoryID, currentID string) (prev, next *AdjacentProduct, err error)
}
