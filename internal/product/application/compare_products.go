package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

// CompareProducts fetches 2-4 published products (with attribute values
// attached) for a side-by-side comparison table. All products must share the
// same category — each category has its own, largely disjoint spec
// vocabulary (see docs/product-v2-design.md's attribute design notes), so
// comparing across categories wouldn't produce a meaningful aligned table.
func CompareProducts(ctx context.Context, repo domain.ProductRepository, ids []string) ([]*domain.ProductWithRelations, error) {
	if len(ids) < 2 || len(ids) > 4 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{
			"ids": {"provide between 2 and 4 product ids to compare"},
		})
	}

	products, err := repo.GetByIDsWithAttributeValues(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(products) != len(ids) {
		return nil, apperr.NewNotFoundError("product")
	}

	categoryID := products[0].CategoryID()
	for _, p := range products {
		if p.CategoryID() != categoryID {
			return nil, apperr.NewValidationError("validation failed", map[string][]string{
				"ids": {"products must be in the same category to compare"},
			})
		}
		if p.Status() != domain.ProductStatusPublished {
			return nil, apperr.NewValidationError("validation failed", map[string][]string{
				"ids": {"only published products can be compared"},
			})
		}
	}

	return products, nil
}
