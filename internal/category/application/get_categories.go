package application

import (
	"context"

	"github.com/trvux/elc-go/internal/category/domain"
)

func GetCategories(ctx context.Context, repo domain.CategoryRepository, filter domain.CategoryFilter) ([]*domain.CategoryWithRelations, error) {
	return repo.GetAll(ctx, filter)
}

func CountCategories(ctx context.Context, repo domain.CategoryRepository, filter domain.CategoryFilter) (int, error) {
	return repo.Count(ctx, filter)
}

func GetCategoryByID(ctx context.Context, repo domain.CategoryRepository, id string) (*domain.CategoryWithRelations, error) {
	return repo.GetByID(ctx, id)
}

func GetCategoryBySlug(ctx context.Context, repo domain.CategoryRepository, slug string) (*domain.CategoryWithRelations, error) {
	return repo.GetBySlug(ctx, slug)
}
