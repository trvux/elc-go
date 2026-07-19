package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

// SubmitProductForReview moves a draft into proposed, awaiting an
// owner/admin's Approve or Reject — see domain.Product.SubmitForReview.
func SubmitProductForReview(ctx context.Context, repo domain.ProductRepository, id string) (*domain.Product, error) {
	existing, err := repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("product")
	}
	product := existing.Product

	if err := product.SubmitForReview(); err != nil {
		return nil, err
	}

	return repo.Update(ctx, product, nil, nil, nil, nil)
}
