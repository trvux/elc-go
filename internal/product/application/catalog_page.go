package application

import (
	"context"

	"github.com/trvux/elc-go/internal/product/domain"
)

func GetCatalogPage(ctx context.Context, repo domain.CatalogPageRepository) (*domain.CatalogPage, error) {
	return repo.Get(ctx)
}

func UpdateCatalogPage(ctx context.Context, repo domain.CatalogPageRepository, input domain.UpdateCatalogPageInput) (*domain.CatalogPage, error) {
	return repo.Update(ctx, input)
}
