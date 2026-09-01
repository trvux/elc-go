package application

import (
	"context"
	"fmt"

	attributedomain "github.com/trvux/elc-go/internal/attribute/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

func CreateProduct(ctx context.Context, repo domain.ProductRepository, attributeRepo attributedomain.AttributeDefinitionRepository, input domain.CreateProductInput) (*domain.Product, error) {
	product, err := domain.NewProduct(
		input.CategoryID, input.BrandID, input.Name, input.Slug,
		input.Description,
		input.Images,
		input.IsFeatured, input.OrderIndex,
		input.MetaTitle, input.MetaDescription,
		input.ProductLineID,
		input.Highlights,
	)
	if err != nil {
		return nil, err
	}

	variants, err := resolveDefaultVariant(input.Variants)
	if err != nil {
		return nil, err
	}
	if err := validateVariantPrices(variants); err != nil {
		return nil, err
	}

	if err := validateAttributeValues(ctx, attributeRepo, input.CategoryID, input.AttributeValues); err != nil {
		return nil, err
	}

	return repo.Create(ctx, product, input.TagIDs, input.Options, variants, input.AttributeValues)
}

// resolveDefaultVariant applies the "exactly one default variant" rule
// before persistence: a product with zero variants is rejected — every
// product must have at least one real variant (Product itself carries no
// sku/mpn/price, see the domain.Product doc comment). A single-variant
// product is auto-defaulted regardless of what the caller sent (rejected
// afterward if that one variant is component-only — see below); multiple
// variants with none marked default auto-default the first STANDALONE one
// (skipping component-only variants, which can't stand in for a product's
// display price — see below), not blindly index 0; multiple variants
// explicitly marked default is a validation error the DB's partial unique
// index would otherwise surface as an opaque constraint violation.
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
		// Auto-default the first variant that CAN legally be default (see
		// the IsComponentOnly check below) rather than blindly variants[0]
		// — a request like [componentOnly, standalone, standalone] with
		// none marked default has a perfectly good standalone candidate;
		// picking index 0 unconditionally would reject it for no reason.
		defaultIdx = 0
		for i, v := range variants {
			if !v.IsComponentOnly {
				defaultIdx = i
				break
			}
		}
		variants[defaultIdx].IsDefault = true
	} else if len(variants) == 1 {
		variants[defaultIdx].IsDefault = true
	}
	// A component-only variant (IsComponentOnly, i.e. is_standalone=false)
	// only exists as part of a bundle and is excluded from
	// RecomputeDisplayCache's default-variant lookup (see
	// infrastructure/variant_repository.go's dv CTE, which filters
	// is_standalone = true) — marking one default would leave the whole
	// product's display_price/default_variant_id/display_stock_status
	// silently NULL despite having other active variants. This only ever
	// fires now for an explicit caller-supplied default (defaultCount==1)
	// or a single-variant product whose only variant is component-only
	// (genuinely unresolvable — nothing else to pick) — the defaultCount==0
	// multi-variant path above already avoids ever landing here when a
	// standalone alternative exists.
	if variants[defaultIdx].IsComponentOnly {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{
			"variants": {"a component-only variant (is_component_only) cannot be marked as default"},
		})
	}
	return variants, nil
}

// validateVariantPrices rejects a price no real retail product can have —
// previously unenforced anywhere (see
// docs/rfc/2026-08-18-product-data-anomaly-detection.md §2.1): a customer-
// facing price of 0 or less always means bad data entry, never a genuine
// price, and this is the data internal/ai's search_products tool grounds
// its answers in. Doesn't check SalePrice against OriginalPrice or
// anything else — out of this RFC's scope.
func validateVariantPrices(variants []domain.ProductVariantInput) error {
	for i, v := range variants {
		if v.OriginalPrice <= 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{
				"variants": {fmt.Sprintf("variant %d: original price must be greater than 0", i)},
			})
		}
		if v.SalePrice != nil && *v.SalePrice <= 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{
				"variants": {fmt.Sprintf("variant %d: sale price must be greater than 0", i)},
			})
		}
	}
	return nil
}
