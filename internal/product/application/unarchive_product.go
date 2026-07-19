package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

// UnarchiveProduct re-lists a discontinued product back to published
// without re-review — see domain.Product.Unarchive. Callers must gate this
// behind authdomain.CanPublishContent, same as ApproveProduct.
func UnarchiveProduct(ctx context.Context, repo domain.ProductRepository, id string) (*domain.Product, error) {
	existing, err := repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("product")
	}
	product := existing.Product

	if err := product.Unarchive(); err != nil {
		return nil, err
	}

	return repo.Update(ctx, product, nil, nil, nil, nil)
}
