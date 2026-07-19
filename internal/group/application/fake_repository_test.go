package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/group/domain"
)

type fakeGroupRepository struct {
	items map[string]*domain.Group
}

func newFakeGroupRepository() *fakeGroupRepository {
	return &fakeGroupRepository{items: map[string]*domain.Group{}}
}

func (r *fakeGroupRepository) GetAll(ctx context.Context, filter domain.GroupFilter) ([]*domain.Group, error) {
	result := make([]*domain.Group, 0, len(r.items))
	for _, g := range r.items {
		if !filter.IncludeDeleted && g.IsDeleted() {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(g.Name()), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, g)
	}
	return result, nil
}

func (r *fakeGroupRepository) GetByID(ctx context.Context, id string) (*domain.Group, error) {
	g, ok := r.items[id]
	if !ok || g.IsDeleted() {
		return nil, nil
	}
	return g, nil
}

func (r *fakeGroupRepository) GetBySlug(ctx context.Context, slug string) (*domain.Group, error) {
	for _, g := range r.items {
		if g.Slug() == slug && !g.IsDeleted() {
			return g, nil
		}
	}
	return nil, nil
}

func (r *fakeGroupRepository) Create(ctx context.Context, group *domain.Group) (*domain.Group, error) {
	for id, g := range r.items {
		if g.Slug() == group.Slug() && g.IsDeleted() {
			// Resurrect!
			resurrected := domain.RehydrateGroup(
				id, group.Name(), group.Slug(), group.ImageURL(),
				group.MetaTitle(), group.MetaDescription(),
				group.IsFeatured(), group.IsHidden(), group.OrderIndex(), group.Content(),
				g.CreatedAt(), time.Now(), nil,
			)
			r.items[id] = resurrected
			return resurrected, nil
		}
	}

	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateGroup(
		id, group.Name(), group.Slug(), group.ImageURL(),
		group.MetaTitle(), group.MetaDescription(),
		group.IsFeatured(), group.IsHidden(), group.OrderIndex(), group.Content(),
		now, now, nil,
	)
	r.items[id] = created
	return created, nil
}

func (r *fakeGroupRepository) Update(ctx context.Context, group *domain.Group) (*domain.Group, error) {
	r.items[group.ID()] = group
	return group, nil
}

func (r *fakeGroupRepository) SoftDelete(ctx context.Context, id string) error {
	g, ok := r.items[id]
	if !ok {
		return nil
	}
	g.MarkDeleted(time.Now())
	return nil
}

func (r *fakeGroupRepository) Restore(ctx context.Context, id string) error {
	g, ok := r.items[id]
	if !ok {
		return nil
	}
	g.Restore()
	return nil
}
