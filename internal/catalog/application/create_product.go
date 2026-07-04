package application

import (
	"context"

	"github.com/trvux/elc-go/internal/catalog/domain"
)

// CreateProduct reconciles pricing and derives normalized_specs BEFORE
// constructing the entity — both NormalizeProductPrice and
// NormalizeProductSpecs are pure domain functions, called here (the
// application layer) rather than inside domain.NewProduct, so the domain
// constructor stays a simple "validate + store what I'm given" like
// brand/service's. This is also where the old TS read-time behavior
// (normalizeProductPrice applied in searchProducts.ts on every request) gets
// intentionally moved to write-time — see docs/catalog.md.
func CreateProduct(ctx context.Context, repo domain.ProductRepository, input domain.CreateProductInput) (*domain.Product, error) {
	var salePriceIn int64
	if input.SalePrice != nil {
		salePriceIn = *input.SalePrice
	}
	originalPrice, salePrice, discountPercent := domain.NormalizeProductPrice(
		input.OriginalPrice, salePriceIn, input.DiscountPercent,
	)

	normalizedSpecs := domain.NormalizeProductSpecs(input.Name, input.Specs)

	product, err := domain.NewProduct(
		input.CategoryID, input.BrandID, input.Name, input.SKU, input.Slug,
		input.Description, input.Specs, normalizedSpecs,
		input.Images, input.Labels,
		originalPrice, &salePrice, discountPercent,
		input.IsFeatured, input.IsPublished, input.OrderIndex,
		input.StockStatus, input.Condition,
		input.MetaTitle, input.MetaDescription, input.MPN, input.GTIN,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, product)
}
