package application

import (
	"context"

	"github.com/trvux/elc-go/internal/project/domain"
)

func GetProjects(ctx context.Context, repo domain.ProjectRepository, filter domain.ProjectFilter) ([]*domain.ProjectWithRelations, error) {
	return repo.GetAll(ctx, filter)
}

func CountProjects(ctx context.Context, repo domain.ProjectRepository, filter domain.ProjectFilter) (int, error) {
	return repo.Count(ctx, filter)
}

func GetProjectByID(ctx context.Context, repo domain.ProjectRepository, id string) (*domain.ProjectWithRelations, error) {
	return repo.GetByID(ctx, id)
}

func GetProjectBySlug(ctx context.Context, repo domain.ProjectRepository, slug string) (*domain.ProjectWithRelations, error) {
	return repo.GetBySlug(ctx, slug)
}

// GetAdjacentProjects delegates the actual sibling lookup + fallback
// straight to the repository (see domain/repository.go's GetAdjacent) — the
// old TS application/getAdjacentProjects.ts did this sorting itself after
// loading every published project into memory; that's now one or two
// indexed SQL queries, so there's nothing left for this layer to orchestrate.
func GetAdjacentProjects(ctx context.Context, repo domain.ProjectRepository, projectTypeID *string, currentID string) (prev, next *domain.AdjacentProject, err error) {
	return repo.GetAdjacent(ctx, projectTypeID, currentID)
}

func GetCategoriesByProjectTypeID(ctx context.Context, repo domain.ProjectRepository, projectTypeID string) ([]domain.CategoryRef, error) {
	return repo.GetCategoriesByProjectTypeID(ctx, projectTypeID)
}
