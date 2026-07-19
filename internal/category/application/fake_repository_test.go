package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/category/domain"
)

type fakeCategoryRepository struct {
	items map[string]*domain.Category
}

func newFakeCategoryRepository() *fakeCategoryRepository {
	return &fakeCategoryRepository{items: map[string]*domain.Category{}}
}

func withRelations(c *domain.Category) *domain.CategoryWithRelations {
	if c == nil {
		return nil
	}
	return &domain.CategoryWithRelations{Category: c}
}

func (r *fakeCategoryRepository) GetAll(ctx context.Context, filter domain.CategoryFilter) ([]*domain.CategoryWithRelations, error) {
	result := make([]*domain.CategoryWithRelations, 0, len(r.items))
	for _, c := range r.items {
		if !filter.IncludeDeleted && c.IsDeleted() {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(c.Name()), strings.ToLower(filter.Search)) {
			continue
		}
		if filter.GroupID != "" && (c.GroupID() == nil || *c.GroupID() != filter.GroupID) {
			continue
		}
		result = append(result, withRelations(c))
	}
	return result, nil
}

func (r *fakeCategoryRepository) Count(ctx context.Context, filter domain.CategoryFilter) (int, error) {
	all, _ := r.GetAll(ctx, filter)
	return len(all), nil
}

func (r *fakeCategoryRepository) GetByID(ctx context.Context, id string) (*domain.CategoryWithRelations, error) {
	c, ok := r.items[id]
	if !ok || c.IsDeleted() {
		return nil, nil
	}
	return withRelations(c), nil
}

func (r *fakeCategoryRepository) GetBySlug(ctx context.Context, slug string) (*domain.CategoryWithRelations, error) {
	for _, c := range r.items {
		if c.Slug() == slug && !c.IsDeleted() {
			return withRelations(c), nil
		}
	}
	return nil, nil
}

func (r *fakeCategoryRepository) Create(ctx context.Context, category *domain.Category) (*domain.Category, error) {
	for id, c := range r.items {
		if c.Slug() == category.Slug() && c.IsDeleted() {
			// Resurrect!
			resurrected := domain.RehydrateCategory(
				id, category.Name(), category.Slug(), category.GroupID(), category.ImageURL(),
				category.MetaTitle(), category.MetaDescription(),
				category.IsFeatured(), category.IsHidden(), category.OrderIndex(), category.Content(),
				c.CreatedAt(), time.Now(), nil,
			)
			r.items[id] = resurrected
			return resurrected, nil
		}
	}

	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateCategory(
		id, category.Name(), category.Slug(), category.GroupID(), category.ImageURL(),
		category.MetaTitle(), category.MetaDescription(),
		category.IsFeatured(), category.IsHidden(), category.OrderIndex(), category.Content(),
		now, now, nil,
	)
	r.items[id] = created
	return created, nil
}

func (r *fakeCategoryRepository) Update(ctx context.Context, category *domain.Category) (*domain.Category, error) {
	r.items[category.ID()] = category
	return category, nil
}

func (r *fakeCategoryRepository) SoftDelete(ctx context.Context, id string) error {
	c, ok := r.items[id]
	if !ok {
		return nil
	}
	c.MarkDeleted(time.Now())
	return nil
}

func (r *fakeCategoryRepository) Restore(ctx context.Context, id string) error {
	c, ok := r.items[id]
	if !ok {
		return nil
	}
	c.Restore()
	return nil
}
