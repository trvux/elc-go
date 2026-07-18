package application

import (
	"context"

	attributedomain "github.com/trvux/elc-go/internal/attribute/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

func UpdateProduct(ctx context.Context, repo domain.ProductRepository, attributeRepo attributedomain.AttributeDefinitionRepository, input domain.UpdateProductInput) (*domain.Product, error) {
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

	if input.Images != nil {
		product.UpdateImages(input.Images)
	}

	if input.IsFeatured != nil {
		product.SetFeatured(*input.IsFeatured)
	}
	if input.OrderIndex != nil {
		product.Reorder(*input.OrderIndex)
	}
	if input.MetaTitle != nil {
		product.UpdateMetaTitle(input.MetaTitle)
	}
	if input.MetaDescription != nil {
		product.UpdateMetaDescription(input.MetaDescription)
	}
	if input.ProductLineID != nil {
		product.UpdateProductLineID(input.ProductLineID)
	}
	if input.ShortDescription != nil {
		product.UpdateShortDescription(input.ShortDescription)
	}

	var variants *[]domain.ProductVariantInput
	if input.Variants != nil {
		resolved, err := resolveDefaultVariant(*input.Variants)
		if err != nil {
			return nil, err
		}
		variants = &resolved
	}

	// Re-validate against the effective attribute set even when this
	// request doesn't resend AttributeValues (nil = "leave untouched") —
	// the category may have changed, or its required-attribute rules may
	// have changed since this product was last saved.
	effectiveAttributeValues := input.AttributeValues
	if effectiveAttributeValues == nil {
		converted := attributeValueRefsToInputs(existing.AttributeValues)
		effectiveAttributeValues = &converted
	}
	if err := validateAttributeValues(ctx, attributeRepo, product.CategoryID(), *effectiveAttributeValues); err != nil {
		return nil, err
	}

	return repo.Update(ctx, product, input.TagIDs, input.Options, variants, input.AttributeValues)
}
