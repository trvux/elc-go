package application

import (
	"context"

	"github.com/trvux/elc-go/internal/catalog/domain"
)

func RestoreProduct(ctx context.Context, repo domain.ProductRepository, id string) error {
	return repo.Restore(ctx, id)
}
