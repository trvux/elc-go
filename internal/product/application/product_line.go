package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

func ListProductLines(ctx context.Context, repo domain.ProductLineRepository, brandID *string, includeDeleted bool) ([]*domain.ProductLine, error) {
	return repo.List(ctx, brandID, includeDeleted)
}

func GetProductLine(ctx context.Context, repo domain.ProductLineRepository, id string) (*domain.ProductLine, error) {
	return repo.GetByID(ctx, id)
}

func CreateProductLine(ctx context.Context, repo domain.ProductLineRepository, input domain.CreateProductLineInput) (*domain.ProductLine, error) {
	line, err := domain.NewProductLine(&input.BrandID, input.CategoryID, input.Code, input.Name, input.TierRank, input.Description, input.MpnPrefixes)
	if err != nil {
		return nil, err
	}
	return repo.Create(ctx, line)
}

func UpdateProductLine(ctx context.Context, repo domain.ProductLineRepository, input domain.UpdateProductLineInput) (*domain.ProductLine, error) {
	existing, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("product line")
	}
	if err := existing.Update(input.CategoryID, input.Name, input.TierRank, input.Description, input.MpnPrefixes); err != nil {
		return nil, err
	}
	return repo.Update(ctx, existing)
}

func DeleteProductLine(ctx context.Context, repo domain.ProductLineRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreProductLine(ctx context.Context, repo domain.ProductLineRepository, id string) error {
	return repo.Restore(ctx, id)
}
