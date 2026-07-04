package application

import (
	"context"

	"github.com/trvux/elc-go/internal/catalog/domain"
)

func GetProductByID(ctx context.Context, repo domain.ProductRepository, id string) (*domain.ProductWithRelations, error) {
	return repo.GetByID(ctx, id)
}

func GetProductBySlug(ctx context.Context, repo domain.ProductRepository, slug string) (*domain.ProductWithRelations, error) {
	return repo.GetBySlug(ctx, slug)
}

func GetProductsByIDs(ctx context.Context, repo domain.ProductRepository, ids []string) ([]*domain.ProductWithRelations, error) {
	return repo.GetByIDs(ctx, ids)
}
