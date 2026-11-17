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
		input.TitleAlign,
		input.Description,
		input.Images,
		input.IsFeatured,
		input.IsPublished,
		input.MetaTitle,
		input.MetaDescription,
		input.OrderIndex,
		input.ProjectTypeID,
		input.ClientName,
		input.Location,
		input.CompletedAt,
		input.TestimonialQuote,
		input.TestimonialAuthor,
		input.Facts,
	)
	if err != nil {
		return nil, err
	}

	if errs := validateCategoryConditions(input.Categories); len(errs) > 0 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"categories": errs})
	}

	if project.IsPublished() {
		if err := publishGate(project, len(input.ServiceIDs), len(input.Categories)); err != nil {
			return nil, err
		}
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

// publishGate turns domain.PublishGaps into a validation error. It runs only
// when a project becomes published (create-as-published, or draft ->
// published), never on edits to an already published row — see PublishGaps.
func publishGate(p *domain.Project, serviceCount, categoryCount int) error {
	gaps := domain.PublishGaps(p, serviceCount, categoryCount)
	if len(gaps) == 0 {
		return nil
	}
	return apperr.NewValidationError("project is not ready to publish", map[string][]string{"publish": gaps})
}
