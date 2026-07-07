package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/project/domain"
)

func CreateProject(ctx context.Context, repo domain.ProjectRepository, input domain.CreateProjectInput) (*domain.Project, error) {
	project, err := domain.NewProject(
		input.Title,
		input.Slug,
		input.Description,
		input.Images,
		input.IsFeatured,
		input.IsPublished,
		input.MetaTitle,
		input.MetaDescription,
		input.Seo,
		input.OrderIndex,
		input.CategoryID,
		input.ProjectTypeID,
	)
	if err != nil {
		return nil, err
	}

	if errs := validateCategoryConditions(input.Categories); len(errs) > 0 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"categories": errs})
	}

	return repo.Create(ctx, project, input.Categories, input.ServiceIDs)
}

func validateCategoryConditions(categories []domain.CategoryCondition) []string {
	var errs []string
	for _, c := range categories {
		errs = append(errs, domain.ValidateCondition(c.Condition)...)
	}
	return errs
}
