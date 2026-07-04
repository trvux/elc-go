package application

import (
	"context"

	"github.com/trvux/elc-go/internal/catalog/domain"
)

// DeleteProduct soft-deletes only — slug_registry cleanup for the deleted
// slug happens automatically via the pre-existing trg_product_slug_registry
// trigger, no Go code needed (see docs/catalog.md).
func DeleteProduct(ctx context.Context, repo domain.ProductRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}
