package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/brand/domain"
)

type fakeBrandRepository struct {
	items map[string]*domain.Brand
}

func newFakeBrandRepository() *fakeBrandRepository {
	return &fakeBrandRepository{items: map[string]*domain.Brand{}}
}

func (r *fakeBrandRepository) GetAll(ctx context.Context, filter domain.BrandFilter) ([]*domain.Brand, error) {
	result := make([]*domain.Brand, 0, len(r.items))
	for _, b := range r.items {
		if !filter.IncludeDeleted && b.IsDeleted() {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(b.Name()), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, b)
	}
	return result, nil
}

func (r *fakeBrandRepository) GetByID(ctx context.Context, id string) (*domain.Brand, error) {
	b, ok := r.items[id]
	if !ok || b.IsDeleted() {
		return nil, nil
	}
	return b, nil
}

func (r *fakeBrandRepository) GetBySlug(ctx context.Context, slug string) (*domain.Brand, error) {
	for _, b := range r.items {
		if b.Slug() == slug && !b.IsDeleted() {
			return b, nil
		}
	}
	return nil, nil
}

func (r *fakeBrandRepository) Create(ctx context.Context, brand *domain.Brand) (*domain.Brand, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateBrand(
		id, brand.Name(), brand.Slug(), brand.LogoURL(),
		brand.MetaTitle(), brand.MetaDescription(),
		brand.IsFeatured(), brand.OrderIndex(), brand.Content(), brand.FAQ(),
		now, now, nil,
	)
	r.items[id] = created
	return created, nil
}

func (r *fakeBrandRepository) Update(ctx context.Context, brand *domain.Brand) (*domain.Brand, error) {
	r.items[brand.ID()] = brand
	return brand, nil
}

func (r *fakeBrandRepository) SoftDelete(ctx context.Context, id string) error {
	b, ok := r.items[id]
	if !ok {
		return nil
	}
	b.MarkDeleted(time.Now())
	return nil
}

func (r *fakeBrandRepository) Restore(ctx context.Context, id string) error {
	b, ok := r.items[id]
	if !ok {
		return nil
	}
	b.Restore()
	return nil
}
