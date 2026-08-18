package application

import (
	"context"

	"github.com/trvux/elc-go/internal/brand/domain"
)

// DeleteBrand soft-deletes the brand row only — it does not cascade into
// products.brand_id (see SoftDelete's comment in the postgres repository).
func DeleteBrand(ctx context.Context, repo domain.BrandRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreBrand(ctx context.Context, repo domain.BrandRepository, id string) error {
	return repo.Restore(ctx, id)
}
