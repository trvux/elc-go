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
	if input.ProjectTypeID != nil {
		project.UpdateProjectTypeID(input.ProjectTypeID)
	}
	if input.ClientName != nil {
		project.UpdateClientName(*input.ClientName)
	}
	if input.Location != nil {
		project.UpdateLocation(*input.Location)
	}
	if input.CompletedAt != nil {
		project.UpdateCompletedAt(input.CompletedAt)
	}
	if input.TestimonialQuote != nil || input.TestimonialAuthor != nil {
		quote := project.TestimonialQuote()
		if input.TestimonialQuote != nil {
			quote = *input.TestimonialQuote
		}
		author := project.TestimonialAuthor()
		if input.TestimonialAuthor != nil {
			author = *input.TestimonialAuthor
		}
		project.UpdateTestimonial(quote, author)
	}

	if input.Categories != nil {
		if errs := validateCategoryConditions(*input.Categories); len(errs) > 0 {
			return nil, apperr.NewValidationError("validation failed", map[string][]string{"categories": errs})
		}
	}

	return repo.Update(ctx, project, input.Categories, input.ServiceIDs, input.TagIDs)
}
