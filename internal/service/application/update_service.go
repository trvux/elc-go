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

	updated, err := repo.Update(ctx, service)
	if err != nil {
		return nil, err
	}
	if input.CategoryIDs != nil {
		if err := repo.SetCategories(ctx, updated.ID(), input.CategoryIDs); err != nil {
			return nil, err
		}
	}
	return updated, nil
}
