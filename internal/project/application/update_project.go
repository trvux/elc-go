package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/project/domain"
)

func UpdateProject(ctx context.Context, repo domain.ProjectRepository, input domain.UpdateProjectInput) (*domain.Project, error) {
	existing, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("project")
	}
	project := existing.Project

	if input.Title != nil {
		if err := project.UpdateTitle(*input.Title); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := project.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.Description != nil {
		project.UpdateDescription(input.Description)
	}
	if input.Images != nil {
		project.UpdateImages(input.Images)
	}
	if input.IsFeatured != nil {
		project.SetFeatured(*input.IsFeatured)
	}
	if input.IsPublished != nil {
		project.SetPublished(*input.IsPublished)
	}
	if input.MetaTitle != nil {
		project.UpdateMetaTitle(input.MetaTitle)
	}
	if input.MetaDescription != nil {
		project.UpdateMetaDescription(input.MetaDescription)
	}
	if input.OrderIndex != nil {
		project.Reorder(*input.OrderIndex)
	}
	if input.CategoryID != nil {
		if err := project.UpdateCategoryID(*input.CategoryID); err != nil {
			return nil, err
		}
	}
	if input.ProjectTypeID != nil {
		project.UpdateProjectTypeID(input.ProjectTypeID)
	}

	if input.Categories != nil {
		if errs := validateCategoryConditions(*input.Categories); len(errs) > 0 {
			return nil, apperr.NewValidationError("validation failed", map[string][]string{"categories": errs})
		}
	}

	return repo.Update(ctx, project, input.Categories, input.ServiceIDs)
}
