package application

import (
	"context"

	"github.com/trvux/elc-go/internal/service-group/domain"
)

func CreateServiceGroup(ctx context.Context, repo domain.ServiceGroupRepository, input domain.CreateServiceGroupInput) (*domain.ServiceGroup, error) {
	serviceGroup, err := domain.NewServiceGroup(
		input.Name,
		input.Slug,
		input.ImageURL,
		input.MetaTitle,
		input.MetaDescription,
		input.IsFeatured,
		input.OrderIndex,
		input.CategoryIDs,
	)
	if err != nil {
		return nil, err
	}

	// repo.Create handles the "resurrect a soft-deleted row with the same
	// slug" rule internally (slug is globally unique even for deleted rows) —
	// see docs/service-group.md.
	return repo.Create(ctx, serviceGroup)
}
