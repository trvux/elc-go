package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

func UpdateProduct(ctx context.Context, repo domain.ProductRepository, input domain.UpdateProductInput) (*domain.Product, error) {
	existing, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("product")
	}
	product := existing.Product

	if input.CategoryID != nil {
		if err := product.UpdateCategoryID(*input.CategoryID); err != nil {
			return nil, err
		}
	}
	if input.BrandID != nil {
		if err := product.UpdateBrandID(*input.BrandID); err != nil {
			return nil, err
		}
	}
	if input.Name != nil {
		if err := product.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := product.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.Description != nil {
		product.UpdateDescription(input.Description)
	}

	// Specs and normalizedSpecs are resolved together, whenever EITHER the
	// specs themselves or the (possibly just-updated) name changes —
	// normalizedSpecs is entirely derived from both, so any name-only change
	// still needs a recompute (e.g. renaming to include "2HP" should add a
	// capacity facet even if specs didn't change). Same "resolve unchanged
	// half to its current value" pattern as service's UpdatePricing.
	if input.Specs != nil || input.Name != nil {
		specs := product.Specs()
		if input.Specs != nil {
			specs = input.Specs
		}
		normalizedSpecs := domain.NormalizeProductSpecs(product.Name(), specs)
		product.UpdateSpecs(specs, normalizedSpecs)
	}

	if input.Images != nil {
		product.UpdateImages(input.Images)
	}
	if input.Labels != nil {
		product.SetLabels(input.Labels)
	}

	if input.IsFeatured != nil {
		product.SetFeatured(*input.IsFeatured)
	}
	if input.IsPublished != nil {
		product.SetPublished(*input.IsPublished)
	}
	if input.OrderIndex != nil {
		product.Reorder(*input.OrderIndex)
	}
	if input.Condition != nil {
		if err := product.UpdateCondition(*input.Condition); err != nil {
			return nil, err
		}
	}
	if input.MetaTitle != nil {
		product.UpdateMetaTitle(input.MetaTitle)
	}
	if input.MetaDescription != nil {
		product.UpdateMetaDescription(input.MetaDescription)
	}
	if input.Seo != nil {
		product.UpdateSeo(*input.Seo)
	}
	if input.ProductLineID != nil {
		product.UpdateProductLineID(input.ProductLineID)
	}
	if input.ShortDescription != nil {
		product.UpdateShortDescription(input.ShortDescription)
	}
	if input.WarrantyMonths != nil || input.WarrantyTerms != nil {
		warrantyMonths := product.WarrantyMonths()
		if input.WarrantyMonths != nil {
			warrantyMonths = input.WarrantyMonths
		}
		warrantyTerms := product.WarrantyTerms()
		if input.WarrantyTerms != nil {
			warrantyTerms = input.WarrantyTerms
		}
		product.UpdateWarranty(warrantyMonths, warrantyTerms)
	}

	var variants *[]domain.ProductVariantInput
	if input.Variants != nil {
		resolved, err := resolveDefaultVariant(*input.Variants)
		if err != nil {
			return nil, err
		}
		variants = &resolved
	}

	return repo.Update(ctx, product, input.TagIDs, input.Options, variants, input.AttributeValues)
}
