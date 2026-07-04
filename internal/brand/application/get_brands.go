package application

import (
	"context"

	"github.com/trvux/elc-go/internal/brand/domain"
)

func GetBrands(ctx context.Context, repo domain.BrandRepository, filter domain.BrandFilter) ([]*domain.Brand, error) {
	return repo.GetAll(ctx, filter)
}

func GetBrandByID(ctx context.Context, repo domain.BrandRepository, id string) (*domain.Brand, error) {
	return repo.GetByID(ctx, id)
}

func GetBrandBySlug(ctx context.Context, repo domain.BrandRepository, slug string) (*domain.Brand, error) {
	return repo.GetBySlug(ctx, slug)
}
