package application

import (
	"context"

	"github.com/trvux/elc-go/internal/category/domain"
)

func DeleteCategory(ctx context.Context, repo domain.CategoryRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreCategory(ctx context.Context, repo domain.CategoryRepository, id string) error {
	return repo.Restore(ctx, id)
}
