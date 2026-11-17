package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/project/domain"
)

func UpdateProjectOrder(ctx context.Context, repo domain.ProjectRepository, id string, orderIndex int) error {
	return repo.UpdateOrder(ctx, id, orderIndex)
}

func ToggleProjectPublish(ctx context.Context, repo domain.ProjectRepository, id string, isPublished bool) error {
	if isPublished {
		existing, err := repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return apperr.NewNotFoundError("project")
		}
		// Already published: nothing to gate, keep the toggle idempotent.
		if !existing.IsPublished() {
			if err := publishGate(existing.Project, len(existing.Services), len(existing.Categories)); err != nil {
				return err
			}
		}
	}
	return repo.TogglePublish(ctx, id, isPublished)
}

func ToggleProjectFeatured(ctx context.Context, repo domain.ProjectRepository, id string, isFeatured bool) error {
	return repo.ToggleFeatured(ctx, id, isFeatured)
}
