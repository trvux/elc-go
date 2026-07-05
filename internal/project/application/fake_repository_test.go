package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/project/domain"
)

// fakeProjectRepository is an in-memory stand-in used only to exercise
// application-layer orchestration (validation, partial update, relation
// pass-through semantics). It deliberately does NOT attempt to replicate the
// real SQL joins/pricing — that's covered by the integration tests against
// the real DB in internal/project/infrastructure.
type fakeProjectRepository struct {
	items      map[string]*domain.Project
	categories map[string][]domain.CategoryCondition
	serviceIDs map[string][]string
}

func newFakeProjectRepository() *fakeProjectRepository {
	return &fakeProjectRepository{
		items:      map[string]*domain.Project{},
		categories: map[string][]domain.CategoryCondition{},
		serviceIDs: map[string][]string{},
	}
}

func (r *fakeProjectRepository) withRelations(p *domain.Project) *domain.ProjectWithRelations {
	var cats []domain.ProjectCategory
	for _, cc := range r.categories[p.ID()] {
		cats = append(cats, domain.ProjectCategory{ID: cc.CategoryID, Condition: cc.Condition})
	}
	var services []domain.ProjectServiceRef
	for _, id := range r.serviceIDs[p.ID()] {
		services = append(services, domain.ProjectServiceRef{ID: id})
	}
	return &domain.ProjectWithRelations{Project: p, Categories: cats, Services: services}
}

func (r *fakeProjectRepository) GetAll(ctx context.Context, filter domain.ProjectFilter) ([]*domain.ProjectWithRelations, error) {
	var result []*domain.ProjectWithRelations
	for _, p := range r.items {
		if !filter.IncludeDeleted && p.IsDeleted() {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(p.Title()), strings.ToLower(filter.Search)) {
			continue
		}
		if filter.CategoryID != nil && p.CategoryID() != *filter.CategoryID {
			continue
		}
		if filter.ExcludeID != nil && p.ID() == *filter.ExcludeID {
			continue
		}
		result = append(result, r.withRelations(p))
	}
	return result, nil
}

func (r *fakeProjectRepository) Count(ctx context.Context, filter domain.ProjectFilter) (int, error) {
	all, _ := r.GetAll(ctx, filter)
	return len(all), nil
}

func (r *fakeProjectRepository) GetByID(ctx context.Context, id string) (*domain.ProjectWithRelations, error) {
	p, ok := r.items[id]
	if !ok || p.IsDeleted() {
		return nil, nil
	}
	return r.withRelations(p), nil
}

func (r *fakeProjectRepository) GetBySlug(ctx context.Context, slug string, withPricing bool) (*domain.ProjectWithRelations, error) {
	for _, p := range r.items {
		if p.Slug() == slug && !p.IsDeleted() {
			return r.withRelations(p), nil
		}
	}
	return nil, nil
}

func (r *fakeProjectRepository) Create(ctx context.Context, project *domain.Project, categories []domain.CategoryCondition, serviceIDs []string) (*domain.Project, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateProject(
		id, project.Title(), project.Slug(), project.Description(), project.Images(),
		project.IsFeatured(), project.IsPublished(), project.MetaTitle(), project.MetaDescription(),
		project.OrderIndex(), project.CategoryID(), project.ProjectTypeID(),
		now, now, nil,
	)
	r.items[id] = created
	r.categories[id] = categories
	r.serviceIDs[id] = serviceIDs
	return created, nil
}

func (r *fakeProjectRepository) Update(ctx context.Context, project *domain.Project, categories *[]domain.CategoryCondition, serviceIDs *[]string) (*domain.Project, error) {
	r.items[project.ID()] = project
	if categories != nil {
		r.categories[project.ID()] = *categories
	}
	if serviceIDs != nil {
		r.serviceIDs[project.ID()] = *serviceIDs
	}
	return project, nil
}

func (r *fakeProjectRepository) SoftDelete(ctx context.Context, id string) error {
	p, ok := r.items[id]
	if !ok {
		return nil
	}
	p.MarkDeleted(time.Now())
	return nil
}

func (r *fakeProjectRepository) Restore(ctx context.Context, id string) error {
	p, ok := r.items[id]
	if !ok {
		return nil
	}
	p.Restore()
	return nil
}

func (r *fakeProjectRepository) UpdateOrder(ctx context.Context, id string, orderIndex int) error {
	p, ok := r.items[id]
	if !ok {
		return nil
	}
	p.Reorder(orderIndex)
	return nil
}

func (r *fakeProjectRepository) TogglePublish(ctx context.Context, id string, isPublished bool) error {
	p, ok := r.items[id]
	if !ok {
		return nil
	}
	p.SetPublished(isPublished)
	return nil
}

func (r *fakeProjectRepository) ToggleFeatured(ctx context.Context, id string, isFeatured bool) error {
	p, ok := r.items[id]
	if !ok {
		return nil
	}
	p.SetFeatured(isFeatured)
	return nil
}

func (r *fakeProjectRepository) GetAdjacent(ctx context.Context, projectTypeID *string, currentID string) (*domain.AdjacentProject, *domain.AdjacentProject, error) {
	return nil, nil, nil
}

func (r *fakeProjectRepository) GetCategoriesByProjectTypeID(ctx context.Context, projectTypeID string) ([]domain.CategoryRef, error) {
	return nil, nil
}
