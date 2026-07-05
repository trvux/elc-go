package application

import (
	"context"

	"github.com/trvux/elc-go/internal/project-type/domain"
)

func GetProjectTypes(ctx context.Context, repo domain.ProjectTypeRepository, filter domain.ProjectTypeFilter) ([]*domain.ProjectTypeWithCategories, error) {
	return repo.GetAll(ctx, filter)
}

func CountProjectTypes(ctx context.Context, repo domain.ProjectTypeRepository, filter domain.ProjectTypeFilter) (int, error) {
	return repo.Count(ctx, filter)
}

func GetProjectTypeByID(ctx context.Context, repo domain.ProjectTypeRepository, id string) (*domain.ProjectTypeWithCategories, error) {
	return repo.GetByID(ctx, id)
}
