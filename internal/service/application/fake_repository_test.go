package application

import (
	"context"
	"fmt"
	"time"

	"github.com/trvux/elc-go/internal/service/domain"
)

type fakeServiceRepository struct {
	items map[string]*domain.Service
	// categories records the last SetCategories call per service id; setCalls counts them.
	categories map[string][]string
	setCalls   int
}

func newFakeServiceRepository() *fakeServiceRepository {
	return &fakeServiceRepository{items: map[string]*domain.Service{}, categories: map[string][]string{}}
}

func (r *fakeServiceRepository) Count(ctx context.Context, filter domain.ServiceFilter) (int, error) {
	return len(r.items), nil
}

func (r *fakeServiceRepository) GetAll(ctx context.Context, filter domain.ServiceFilter) ([]*domain.ServiceWithRelations, error) {
	result := make([]*domain.ServiceWithRelations, 0, len(r.items))
	for _, s := range r.items {
		if !filter.IncludeDeleted && s.IsDeleted() {
			continue
		}
		result = append(result, &domain.ServiceWithRelations{Service: s})
	}
	return result, nil
}

func (r *fakeServiceRepository) GetByID(ctx context.Context, id string) (*domain.ServiceWithRelations, error) {
	s, ok := r.items[id]
	if !ok || s.IsDeleted() {
		return nil, nil
	}
	return &domain.ServiceWithRelations{Service: s}, nil
}

func (r *fakeServiceRepository) GetBySlug(ctx context.Context, slug string) (*domain.ServiceWithRelations, error) {
	for _, s := range r.items {
		if s.Slug() == slug && !s.IsDeleted() {
			return &domain.ServiceWithRelations{Service: s}, nil
		}
	}
	return nil, nil
}

func (r *fakeServiceRepository) Create(ctx context.Context, service *domain.Service) (*domain.Service, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateService(
		id, service.Title(), service.TitleAlign(), service.Slug(),
		service.Labels(), service.Line(), service.BriefPrompts(), service.BriefFields(),
		service.Description(), service.Content(),
		service.Images(), service.MetaTitle(), service.MetaDescription(),
		service.IsPublished(), service.OrderIndex(),
		now, now, nil,
	)
	r.items[id] = created
	return created, nil
}

func (r *fakeServiceRepository) Update(ctx context.Context, service *domain.Service) (*domain.Service, error) {
	r.items[service.ID()] = service
	return service, nil
}

func (r *fakeServiceRepository) SoftDelete(ctx context.Context, id string) error {
	s, ok := r.items[id]
	if !ok {
		return nil
	}
	s.MarkDeleted(time.Now())
	return nil
}

func (r *fakeServiceRepository) Restore(ctx context.Context, id string) error {
	s, ok := r.items[id]
	if !ok {
		return nil
	}
	s.Restore()
	return nil
}

func (r *fakeServiceRepository) SetCategories(ctx context.Context, serviceID string, categoryIDs []string) error {
	r.setCalls++
	r.categories[serviceID] = categoryIDs
	return nil
}
