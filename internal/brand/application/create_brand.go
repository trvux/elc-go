package application

import (
	"context"

	"github.com/trvux/elc-go/internal/brand/domain"
)

func CreateBrand(ctx context.Context, repo domain.BrandRepository, input domain.CreateBrandInput) (*domain.Brand, error) {
	brand, err := domain.NewBrand(
		input.Name,
		input.Slug,
		input.LogoURL,
		input.MetaTitle,
		input.MetaDescription,
		input.IsFeatured,
		input.OrderIndex,
		input.Content,
		input.WarrantyPolicy,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, brand)
}
