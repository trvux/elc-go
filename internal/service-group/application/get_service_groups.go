package application

import (
	"context"

	"github.com/trvux/elc-go/internal/service-group/domain"
)

func GetServiceGroups(ctx context.Context, repo domain.ServiceGroupRepository, filter domain.ServiceGroupFilter) ([]*domain.ServiceGroup, error) {
	return repo.GetAll(ctx, filter)
}

func GetServiceGroupByID(ctx context.Context, repo domain.ServiceGroupRepository, id string) (*domain.ServiceGroup, error) {
	return repo.GetByID(ctx, id)
}

func GetServiceGroupBySlug(ctx context.Context, repo domain.ServiceGroupRepository, slug string) (*domain.ServiceGroup, error) {
	return repo.GetBySlug(ctx, slug)
}
