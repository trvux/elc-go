package application

import (
	"context"

	"github.com/trvux/elc-go/internal/service/domain"
)

func GetServices(ctx context.Context, repo domain.ServiceRepository, filter domain.ServiceFilter) ([]*domain.ServiceWithRelations, error) {
	return repo.GetAll(ctx, filter)
}

func CountServices(ctx context.Context, repo domain.ServiceRepository, filter domain.ServiceFilter) (int, error) {
	return repo.Count(ctx, filter)
}

func GetServiceByID(ctx context.Context, repo domain.ServiceRepository, id string) (*domain.ServiceWithRelations, error) {
	return repo.GetByID(ctx, id)
}

func GetServiceBySlug(ctx context.Context, repo domain.ServiceRepository, slug string) (*domain.ServiceWithRelations, error) {
	return repo.GetBySlug(ctx, slug)
}
