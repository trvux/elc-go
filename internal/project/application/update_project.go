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
	wasPublished := project.IsPublished()

	if err := project.Update(input); err != nil {
		return nil, err
	}

	if input.Categories != nil {
		if errs := validateCategoryConditions(*input.Categories); len(errs) > 0 {
			return nil, apperr.NewValidationError("validation failed", map[string][]string{"categories": errs})
		}
	}

	// Gate only the draft -> published transition, against the state AFTER
	// this write (relations the request doesn't touch keep their old count).
	if !wasPublished && project.IsPublished() {
		serviceCount, categoryCount := len(existing.Services), len(existing.Categories)
		if input.ServiceIDs != nil {
			serviceCount = len(*input.ServiceIDs)
		}
		if input.Categories != nil {
			categoryCount = len(*input.Categories)
		}
		if err := publishGate(project, serviceCount, categoryCount); err != nil {
			return nil, err
		}
	}

	return repo.Update(ctx, project, input.Categories, input.ServiceIDs)
}
