package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
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

	if err := pt.Update(input); err != nil {
		return nil, err
	}
	if input.IsFeatured != nil {
		pt.SetFeatured(*input.IsFeatured)
	}
	if input.OrderIndex != nil {
		pt.Reorder(*input.OrderIndex)
	}

	return repo.Update(ctx, pt, input.CategoryIDs)
}
