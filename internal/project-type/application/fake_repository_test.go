package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/project-type/domain"
)

// fakeProjectTypeRepository is an in-memory stand-in used only to exercise
// application-layer orchestration (validation, partial update, relation
// pass-through semantics). It deliberately does NOT attempt to replicate the
// real SQL joins — that's covered by the integration tests against the real
// DB in internal/project-type/infrastructure.
type fakeProjectTypeRepository struct {
	items      map[string]*domain.ProjectType
	categories map[string][]string
}

func newFakeProjectTypeRepository() *fakeProjectTypeRepository {
	return &fakeProjectTypeRepository{
		items:      map[string]*domain.ProjectType{},
		categories: map[string][]string{},
	}
}

func (r *fakeProjectTypeRepository) withCategories(p *domain.ProjectType) *domain.ProjectTypeWithCategories {
	var cats []domain.CategoryRef
	for _, id := range r.categories[p.ID()] {
		cats = append(cats, domain.CategoryRef{ID: id})
	}
	return &domain.ProjectTypeWithCategories{ProjectType: p, Categories: cats}
}

func (r *fakeProjectTypeRepository) GetAll(ctx context.Context, filter domain.ProjectTypeFilter) ([]*domain.ProjectTypeWithCategories, error) {
	var result []*domain.ProjectTypeWithCategories
	for _, p := range r.items {
		if !filter.IncludeDeleted && p.IsDeleted() {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(p.Name()), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, r.withCategories(p))
	}
	return result, nil
}

func (r *fakeProjectTypeRepository) Count(ctx context.Context, filter domain.ProjectTypeFilter) (int, error) {
	all, _ := r.GetAll(ctx, filter)
	return len(all), nil
}

func (r *fakeProjectTypeRepository) GetByID(ctx context.Context, id string) (*domain.ProjectTypeWithCategories, error) {
	p, ok := r.items[id]
	if !ok || p.IsDeleted() {
		return nil, nil
	}
	return r.withCategories(p), nil
}

func (r *fakeProjectTypeRepository) Create(ctx context.Context, pt *domain.ProjectType, categoryIDs []string) (*domain.ProjectType, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateProjectType(
		id, pt.Name(), pt.Slug(), pt.Image(), pt.MetaTitle(), pt.MetaDescription(),
		pt.IsFeatured(), pt.OrderIndex(), pt.Content(), now, now, nil,
	)
	r.items[id] = created
	r.categories[id] = categoryIDs
	return created, nil
}

func (r *fakeProjectTypeRepository) Update(ctx context.Context, pt *domain.ProjectType, categoryIDs *[]string) (*domain.ProjectType, error) {
	r.items[pt.ID()] = pt
	if categoryIDs != nil {
		r.categories[pt.ID()] = *categoryIDs
	}
	return pt, nil
}

func (r *fakeProjectTypeRepository) SoftDelete(ctx context.Context, id string) error {
	p, ok := r.items[id]
	if !ok {
		return nil
	}
	p.MarkDeleted(time.Now())
	delete(r.categories, id)
	return nil
}
