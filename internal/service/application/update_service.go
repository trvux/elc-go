package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/service/domain"
)

func UpdateService(ctx context.Context, repo domain.ServiceRepository, input domain.UpdateServiceInput) (*domain.Service, error) {
	existing, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("service")
	}
	service := existing.Service

	if err := service.Update(input); err != nil {
		return nil, err
	}

	return repo.Update(ctx, service)
}
