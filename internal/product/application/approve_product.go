package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

// ApproveProduct publishes a proposed product — see domain.Product.Approve.
// Callers must gate this behind authdomain.CanPublishContent; only an
// owner/admin, not the submitting employee, may call it.
func ApproveProduct(ctx context.Context, repo domain.ProductRepository, id string) (*domain.Product, error) {
	existing, err := repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("product")
	}
	product := existing.Product

	if err := product.Approve(); err != nil {
		return nil, err
	}

	return repo.Update(ctx, product, nil, nil, nil, nil)
}
