package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/service-group/domain"
)

func UpdateServiceGroup(ctx context.Context, repo domain.ServiceGroupRepository, input domain.UpdateServiceGroupInput) (*domain.ServiceGroup, error) {
	serviceGroup, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if serviceGroup == nil {
		return nil, apperr.NewNotFoundError("service group")
	}

	if input.Name != nil {
		if err := serviceGroup.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := serviceGroup.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.ImageURL != nil {
		serviceGroup.UpdateImageURL(input.ImageURL)
	}
	if input.MetaTitle != nil {
		serviceGroup.UpdateMetaTitle(input.MetaTitle)
	}
	if input.MetaDescription != nil {
		serviceGroup.UpdateMetaDescription(input.MetaDescription)
	}
	if input.IsFeatured != nil {
		serviceGroup.SetFeatured(*input.IsFeatured)
	}
	if input.OrderIndex != nil {
		serviceGroup.Reorder(*input.OrderIndex)
	}
	if input.CategoryIDs != nil {
		serviceGroup.SetCategoryIDs(input.CategoryIDs)
	}

	return repo.Update(ctx, serviceGroup)
}
