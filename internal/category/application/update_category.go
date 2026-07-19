package application

import (
	"context"

	"github.com/trvux/elc-go/internal/category/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdateCategory(ctx context.Context, repo domain.CategoryRepository, input domain.UpdateCategoryInput) (*domain.Category, error) {
	withRelations, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if withRelations == nil {
		return nil, apperr.NewNotFoundError("category")
	}
	c := withRelations.Category

	if input.Name != nil {
		if err := c.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := c.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.GroupID != nil {
		c.UpdateGroupID(input.GroupID)
	}
	if input.ImageURL != nil {
		c.UpdateImageURL(input.ImageURL)
	}
	if input.MetaTitle != nil {
		c.UpdateMetaTitle(input.MetaTitle)
	}
	if input.MetaDescription != nil {
		c.UpdateMetaDescription(input.MetaDescription)
	}
	if input.IsFeatured != nil {
		c.SetFeatured(*input.IsFeatured)
	}
	if input.IsHidden != nil {
		c.SetHidden(*input.IsHidden)
	}
	if input.OrderIndex != nil {
		c.Reorder(*input.OrderIndex)
	}
	if input.Content != nil {
		c.UpdateContent(input.Content)
	}

	return repo.Update(ctx, c)
}
