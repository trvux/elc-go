package application

import (
	"context"

	"github.com/trvux/elc-go/internal/project-type/domain"
)

func CreateProjectType(ctx context.Context, repo domain.ProjectTypeRepository, input domain.CreateProjectTypeInput) (*domain.ProjectType, error) {
	pt, err := domain.NewProjectType(
		input.Name,
		input.Slug,
		input.Image,
		input.MetaTitle,
		input.MetaDescription,
		input.IsFeatured,
		input.OrderIndex,
		input.Content,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, pt, input.CategoryIDs)
}
