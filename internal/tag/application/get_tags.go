package application

import (
	"context"

	"github.com/trvux/elc-go/internal/tag/domain"
)

func GetTags(ctx context.Context, repo domain.TagRepository, filter domain.TagFilter) ([]*domain.Tag, error) {
	return repo.GetAll(ctx, filter)
}

func GetTagByID(ctx context.Context, repo domain.TagRepository, id string) (*domain.Tag, error) {
	return repo.GetByID(ctx, id)
}

func GetTagBySlug(ctx context.Context, repo domain.TagRepository, slug string) (*domain.Tag, error) {
	return repo.GetBySlug(ctx, slug)
}
