package application

import (
	"context"

	"github.com/trvux/elc-go/internal/service/domain"
)

func CreateService(ctx context.Context, repo domain.ServiceRepository, input domain.CreateServiceInput) (*domain.Service, error) {
	service, err := domain.NewService(
		input.Title, input.Slug,
		input.GroupID, input.CategoryID,
		input.OriginalPrice, input.DiscountPercent,
		input.PriceDisplayText, input.Labels, input.Description, input.Content,
		input.Images, input.MetaTitle, input.MetaDescription,
		input.Seo,
		input.IsFeatured, input.IsPublished, input.OrderIndex,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, service)
}
