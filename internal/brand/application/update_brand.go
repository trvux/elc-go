package application

import (
	"context"

	"github.com/trvux/elc-go/internal/brand/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/seo"
)

func UpdateBrand(ctx context.Context, repo domain.BrandRepository, input domain.UpdateBrandInput) (*domain.Brand, error) {
	brand, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, apperr.NewNotFoundError("brand")
	}

	if input.Name != nil {
		if err := brand.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := brand.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.LogoURL != nil {
		brand.UpdateLogoURL(*input.LogoURL)
	}
	if input.MetaTitle != nil && !seo.Unchanged(brand.MetaTitle(), input.MetaTitle) {
		if err := brand.UpdateMetaTitle(input.MetaTitle); err != nil {
			return nil, err
		}
	}
	if input.MetaDescription != nil && !seo.Unchanged(brand.MetaDescription(), input.MetaDescription) {
		if err := brand.UpdateMetaDescription(input.MetaDescription); err != nil {
			return nil, err
		}
	}
	if input.IsFeatured != nil {
		brand.SetFeatured(*input.IsFeatured)
	}
	if input.OrderIndex != nil {
		brand.Reorder(*input.OrderIndex)
	}
	if input.Content != nil {
		brand.UpdateContent(input.Content)
	}
	if input.WarrantyPolicy != nil {
		brand.UpdateWarrantyPolicy(input.WarrantyPolicy)
	}

	return repo.Update(ctx, brand)
}
