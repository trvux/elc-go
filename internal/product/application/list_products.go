package application

import (
	"context"

	"github.com/trvux/elc-go/internal/product/domain"
)

// ListProducts is a thin passthrough to repo.GetAll — this single use case
// replaces BOTH the old getProducts.ts AND searchProducts.ts. "Search" is
// just another field on the same ProductFilter now; the repository/SQL layer
// (full-text search_vector + pg_trgm) handles it entirely, so there is no
// separate search use case the way the old TS code had a whole extra
// Fuse.js-based function. See docs/catalog.md.
func ListProducts(ctx context.Context, repo domain.ProductRepository, filter domain.ProductFilter) (*domain.ProductListResult, error) {
	return repo.GetAll(ctx, filter)
}

func CountProducts(ctx context.Context, repo domain.ProductRepository, filter domain.ProductFilter) (int, error) {
	return repo.Count(ctx, filter)
}
