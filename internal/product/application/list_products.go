package application

import (
	"context"

	"github.com/trvux/elc-go/internal/product/domain"
)

// ListProducts is a thin passthrough to repo.GetAll — bare paginated listing,
// no search/filter/facet logic (removed; see docs/catalog.md history).
func ListProducts(ctx context.Context, repo domain.ProductRepository, filter domain.ProductFilter) (*domain.ProductListResult, error) {
	return repo.GetAll(ctx, filter)
}

func CountProducts(ctx context.Context, repo domain.ProductRepository, filter domain.ProductFilter) (int, error) {
	return repo.Count(ctx, filter)
}
