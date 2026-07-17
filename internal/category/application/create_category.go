package application

import (
	"context"

	"github.com/trvux/elc-go/internal/category/domain"
)

func CreateCategory(ctx context.Context, repo domain.CategoryRepository, input domain.CreateCategoryInput) (*domain.Category, error) {
	c, err := domain.NewCategory(
		input.Name,
		input.Slug,
		input.GroupID,
		input.ImageURL,
		input.MetaTitle,
		input.MetaDescription,
		input.IsFeatured,
		input.OrderIndex,
		input.Content,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, c)
}
