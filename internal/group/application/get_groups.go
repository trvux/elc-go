package application

import (
	"context"

	"github.com/trvux/elc-go/internal/group/domain"
)

func GetGroups(ctx context.Context, repo domain.GroupRepository, filter domain.GroupFilter) ([]*domain.Group, error) {
	return repo.GetAll(ctx, filter)
}

func GetGroupByID(ctx context.Context, repo domain.GroupRepository, id string) (*domain.Group, error) {
	return repo.GetByID(ctx, id)
}

func GetGroupBySlug(ctx context.Context, repo domain.GroupRepository, slug string) (*domain.Group, error) {
	return repo.GetBySlug(ctx, slug)
}
