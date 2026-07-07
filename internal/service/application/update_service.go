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

	if input.Title != nil {
		if err := service.UpdateTitle(*input.Title); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := service.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.GroupID != nil {
		service.UpdateGroupID(input.GroupID)
	}
	if input.CategoryID != nil {
		service.UpdateCategoryID(input.CategoryID)
	}

	// OriginalPrice and DiscountPercent must be resolved together — whichever
	// one isn't part of this update keeps its CURRENT value from `service`,
	// so SalePrice() is always computed from a consistent pair. This is the
	// fix for the old bug where changing only discountPercent left sale_price
	// stale (see docs/service.md).
	if input.OriginalPrice != nil || input.DiscountPercent != nil {
		originalPrice := service.OriginalPrice()
		if input.OriginalPrice != nil {
			originalPrice = input.OriginalPrice
		}
		discountPercent := service.DiscountPercent()
		if input.DiscountPercent != nil {
			discountPercent = input.DiscountPercent
		}
		service.UpdatePricing(originalPrice, discountPercent)
	}

	if input.PriceDisplayText != nil {
		service.UpdatePriceDisplayText(input.PriceDisplayText)
	}
	if input.Labels != nil {
		service.SetLabels(input.Labels)
	}
	if input.Description != nil {
		service.UpdateDescription(input.Description)
	}
	if input.Content != nil {
		service.UpdateContent(input.Content)
	}
	if input.Image != nil {
		service.UpdateImage(input.Image)
	}
	if input.MetaTitle != nil {
		service.UpdateMetaTitle(input.MetaTitle)
	}
	if input.MetaDescription != nil {
		service.UpdateMetaDescription(input.MetaDescription)
	}
	if input.Seo != nil {
		service.UpdateSeo(*input.Seo)
	}
	if input.IsFeatured != nil {
		service.SetFeatured(*input.IsFeatured)
	}
	if input.IsPublished != nil {
		service.SetPublished(*input.IsPublished)
	}
	if input.OrderIndex != nil {
		service.Reorder(*input.OrderIndex)
	}

	return repo.Update(ctx, service)
}
