package application

import (
	"context"

	"github.com/trvux/elc-go/internal/brand/domain"
)

// DeleteBrand soft-deletes — see docs/brand.md for the products.brand_id
// cleanup this cascades into at the repository level.
func DeleteBrand(ctx context.Context, repo domain.BrandRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreBrand(ctx context.Context, repo domain.BrandRepository, id string) error {
	return repo.Restore(ctx, id)
}
