package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

// ArchiveProduct discontinues a published product while keeping its row/URL
// alive — see domain.Product.Archive. Callers must gate this behind
// authdomain.CanPublishContent, same as ApproveProduct.
func ArchiveProduct(ctx context.Context, repo domain.ProductRepository, id string) (*domain.Product, error) {
	existing, err := repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("product")
	}
	product := existing.Product

	if err := product.Archive(); err != nil {
		return nil, err
	}

	return repo.Update(ctx, product, nil, nil, nil, nil)
}
