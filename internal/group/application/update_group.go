package application

import (
	"context"

	"github.com/trvux/elc-go/internal/group/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdateGroup(ctx context.Context, repo domain.GroupRepository, input domain.UpdateGroupInput) (*domain.Group, error) {
	g, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, apperr.NewNotFoundError("group")
	}

	if input.Name != nil {
		if err := g.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := g.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.ImageURL != nil {
		g.UpdateImageURL(input.ImageURL)
	}
	if input.MetaTitle != nil {
		if err := g.UpdateMetaTitle(input.MetaTitle); err != nil {
			return nil, err
		}
	}
	if input.MetaDescription != nil {
		if err := g.UpdateMetaDescription(input.MetaDescription); err != nil {
			return nil, err
		}
	}
	if input.IsFeatured != nil {
		g.SetFeatured(*input.IsFeatured)
	}
	if input.IsHidden != nil {
		g.SetHidden(*input.IsHidden)
	}
	if input.OrderIndex != nil {
		g.Reorder(*input.OrderIndex)
	}
	if input.Content != nil {
		g.UpdateContent(input.Content)
	}

	return repo.Update(ctx, g)
}
