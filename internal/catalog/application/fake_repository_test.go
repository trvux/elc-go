package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/catalog/domain"
)

type fakeProductRepository struct {
	items map[string]*domain.Product
}

func newFakeProductRepository() *fakeProductRepository {
	return &fakeProductRepository{items: map[string]*domain.Product{}}
}

func toWithRelations(p *domain.Product) *domain.ProductWithRelations {
	return &domain.ProductWithRelations{Product: p}
}

func (r *fakeProductRepository) GetAll(ctx context.Context, filter domain.ProductFilter) (*domain.ProductListResult, error) {
	result := make([]*domain.ProductWithRelations, 0, len(r.items))
	for _, p := range r.items {
		if !filter.IncludeDeleted && p.IsDeleted() {
			continue
		}
		if filter.CategoryID != nil && p.CategoryID() != *filter.CategoryID {
			continue
		}
		if filter.BrandID != nil && p.BrandID() != *filter.BrandID {
			continue
		}
		if filter.IsFeatured != nil && p.IsFeatured() != *filter.IsFeatured {
			continue
		}
		if filter.IsPublished != nil && p.IsPublished() != *filter.IsPublished {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(p.Name()), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, toWithRelations(p))
	}
	return &domain.ProductListResult{Products: result, TotalCount: len(result)}, nil
}

func (r *fakeProductRepository) Count(ctx context.Context, filter domain.ProductFilter) (int, error) {
	res, err := r.GetAll(ctx, filter)
	if err != nil {
		return 0, err
	}
	return res.TotalCount, nil
}

func (r *fakeProductRepository) GetByID(ctx context.Context, id string) (*domain.ProductWithRelations, error) {
	p, ok := r.items[id]
	if !ok || p.IsDeleted() {
		return nil, nil
	}
	return toWithRelations(p), nil
}

func (r *fakeProductRepository) GetBySlug(ctx context.Context, slug string) (*domain.ProductWithRelations, error) {
	for _, p := range r.items {
		if p.Slug() == slug && !p.IsDeleted() {
			return toWithRelations(p), nil
		}
	}
	return nil, nil
}

func (r *fakeProductRepository) GetByIDs(ctx context.Context, ids []string) ([]*domain.ProductWithRelations, error) {
	result := make([]*domain.ProductWithRelations, 0, len(ids))
	for _, id := range ids {
		if p, ok := r.items[id]; ok && !p.IsDeleted() {
			result = append(result, toWithRelations(p))
		}
	}
	return result, nil
}

func (r *fakeProductRepository) Create(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateProduct(
		id, product.CategoryID(), product.BrandID(), product.Name(), product.SKU(), product.Slug(),
		product.Description(), product.Specs(), product.NormalizedSpecs(),
		product.Images(), product.Labels(),
		product.OriginalPrice(), product.SalePrice(), product.DiscountPercent(),
		product.IsFeatured(), product.IsPublished(), product.OrderIndex(),
		product.StockStatus(), product.Condition(),
		product.MetaTitle(), product.MetaDescription(), product.MPN(), product.GTIN(),
		product.Seo(),
		now, now, nil,
	)
	r.items[id] = created
	return created, nil
}

func (r *fakeProductRepository) Update(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	r.items[product.ID()] = product
	return product, nil
}

func (r *fakeProductRepository) SoftDelete(ctx context.Context, id string) error {
	p, ok := r.items[id]
	if !ok {
		return nil
	}
	p.MarkDeleted(time.Now())
	return nil
}

func (r *fakeProductRepository) Restore(ctx context.Context, id string) error {
	p, ok := r.items[id]
	if !ok {
		return nil
	}
	p.Restore()
	return nil
}

func (r *fakeProductRepository) GetAdjacent(ctx context.Context, categoryID, currentID string) (*domain.AdjacentProduct, *domain.AdjacentProduct, error) {
	return nil, nil, nil
}
