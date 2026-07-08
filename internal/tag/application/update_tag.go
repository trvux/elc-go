package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/tag/domain"
)

func UpdateTag(ctx context.Context, repo domain.TagRepository, input domain.UpdateTagInput) (*domain.Tag, error) {
	tag, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if tag == nil {
		return nil, apperr.NewNotFoundError("tag")
	}

	if input.Name != nil {
		if err := tag.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := tag.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}

	return repo.Update(ctx, tag)
}
