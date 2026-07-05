package application

import (
	"context"

	"github.com/trvux/elc-go/internal/project/domain"
)

func UpdateProjectOrder(ctx context.Context, repo domain.ProjectRepository, id string, orderIndex int) error {
	return repo.UpdateOrder(ctx, id, orderIndex)
}

func ToggleProjectPublish(ctx context.Context, repo domain.ProjectRepository, id string, isPublished bool) error {
	return repo.TogglePublish(ctx, id, isPublished)
}

func ToggleProjectFeatured(ctx context.Context, repo domain.ProjectRepository, id string, isFeatured bool) error {
	return repo.ToggleFeatured(ctx, id, isFeatured)
}
