package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/seo"
	"github.com/trvux/elc-go/internal/project-type/domain"
)

func UpdateProjectType(ctx context.Context, repo domain.ProjectTypeRepository, input domain.UpdateProjectTypeInput) (*domain.ProjectType, error) {
	withCategories, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if withCategories == nil {
		return nil, apperr.NewNotFoundError("project type")
	}
	pt := withCategories.ProjectType

	if input.Name != nil {
		if err := pt.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := pt.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.Image != nil {
		pt.UpdateImage(input.Image)
	}
	if input.MetaTitle != nil && !seo.Unchanged(pt.MetaTitle(), input.MetaTitle) {
		if err := pt.UpdateMetaTitle(input.MetaTitle); err != nil {
			return nil, err
		}
	}
	if input.MetaDescription != nil && !seo.Unchanged(pt.MetaDescription(), input.MetaDescription) {
		if err := pt.UpdateMetaDescription(input.MetaDescription); err != nil {
			return nil, err
		}
	}
	if input.IsFeatured != nil {
		pt.SetFeatured(*input.IsFeatured)
	}
	if input.OrderIndex != nil {
		pt.Reorder(*input.OrderIndex)
	}

	return repo.Update(ctx, pt, input.CategoryIDs)
}
