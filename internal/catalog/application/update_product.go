package application

import (
	"context"

	"github.com/trvux/elc-go/internal/catalog/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
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
	if input.SKU != nil {
		if err := product.UpdateSKU(*input.SKU); err != nil {
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

	// OriginalPrice/SalePrice/DiscountPercent are resolved together and run
	// back through NormalizeProductPrice — a partial update touching only one
	// of the three must not leave the other two stale or inconsistent, same
	// reasoning as service's UpdatePricing (see docs/service.md), widened
	// here to all three fields since catalog persists all three (service only
	// persists two and derives SalePrice on read).
	if input.OriginalPrice != nil || input.SalePrice != nil || input.DiscountPercent != nil {
		originalPrice := product.OriginalPrice()
		if input.OriginalPrice != nil {
			originalPrice = *input.OriginalPrice
		}
		var salePrice int64
		if product.SalePrice() != nil {
			salePrice = *product.SalePrice()
		}
		if input.SalePrice != nil {
			salePrice = *input.SalePrice
		}
		discountPercent := product.DiscountPercent()
		if input.DiscountPercent != nil {
			discountPercent = *input.DiscountPercent
		}

		normOriginal, normSale, normDiscount := domain.NormalizeProductPrice(originalPrice, salePrice, discountPercent)
		product.UpdatePricing(normOriginal, &normSale, normDiscount)
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
	if input.StockStatus != nil {
		product.UpdateStockStatus(*input.StockStatus)
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
	if input.MPN != nil {
		product.UpdateMPN(input.MPN)
	}
	if input.GTIN != nil {
		product.UpdateGTIN(input.GTIN)
	}

	return repo.Update(ctx, product)
}
