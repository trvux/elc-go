package application

import (
	"context"

	"github.com/trvux/elc-go/internal/tag/domain"
)

func CreateTag(ctx context.Context, repo domain.TagRepository, input domain.CreateTagInput) (*domain.Tag, error) {
	tag, err := domain.NewTag(input.Name, input.Slug)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, tag)
}
