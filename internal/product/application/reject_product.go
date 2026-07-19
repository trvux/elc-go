package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

// RejectProduct sends a proposed product back to draft with a reason — see
// domain.Product.Reject. Callers must gate this behind
// authdomain.CanPublishContent, same as ApproveProduct.
func RejectProduct(ctx context.Context, repo domain.ProductRepository, id string, reason string) (*domain.Product, error) {
	existing, err := repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("product")
	}
	product := existing.Product

	if err := product.Reject(reason); err != nil {
		return nil, err
	}

	return repo.Update(ctx, product, nil, nil, nil, nil)
}
