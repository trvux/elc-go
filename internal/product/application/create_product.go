package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

func CreateProduct(ctx context.Context, repo domain.ProductRepository, input domain.CreateProductInput) (*domain.Product, error) {
	product, err := domain.NewProduct(
		input.CategoryID, input.BrandID, input.Name, input.Slug,
		input.Description,
		input.Images, input.Labels,
		input.IsFeatured, input.IsPublished, input.OrderIndex,
		input.Condition,
		input.MetaTitle, input.MetaDescription,
		input.ProductLineID, input.ShortDescription, input.WarrantyMonths, input.WarrantyTerms,
	)
	if err != nil {
		return nil, err
	}

	variants, err := resolveDefaultVariant(input.Variants)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, product, input.TagIDs, input.Options, variants, input.AttributeValues)
}

// resolveDefaultVariant applies the "exactly one default variant" rule
// before persistence: a product with zero variants is rejected — every
// product must have at least one real variant (Product itself carries no
// sku/mpn/price, see the domain.Product doc comment). A single-variant
// product is auto-defaulted regardless of what the caller sent; multiple
// variants with none marked default default the first one; multiple
// variants explicitly marked default is a validation error the DB's partial
// unique index would otherwise surface as an opaque constraint violation.
func resolveDefaultVariant(variants []domain.ProductVariantInput) ([]domain.ProductVariantInput, error) {
	if len(variants) == 0 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{
			"variants": {"at least one variant is required"},
		})
	}
	defaultCount := 0
	defaultIdx := -1
	for i, v := range variants {
		if v.IsDefault {
			defaultCount++
			defaultIdx = i
		}
	}
	if defaultCount > 1 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{
			"variants": {"only one variant may be marked as default"},
		})
	}
	if defaultCount == 0 {
		variants[0].IsDefault = true
	} else if len(variants) == 1 {
		variants[defaultIdx].IsDefault = true
	}
	return variants, nil
}
