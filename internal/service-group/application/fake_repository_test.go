package application

import (
	"context"
	"fmt"
	"time"

	"github.com/trvux/elc-go/internal/service-group/domain"
)

type fakeServiceGroupRepository struct {
	items map[string]*domain.ServiceGroup
}

func newFakeServiceGroupRepository() *fakeServiceGroupRepository {
	return &fakeServiceGroupRepository{items: map[string]*domain.ServiceGroup{}}
}

func (r *fakeServiceGroupRepository) GetAll(ctx context.Context, filter domain.ServiceGroupFilter) ([]*domain.ServiceGroup, error) {
	result := make([]*domain.ServiceGroup, 0, len(r.items))
	for _, sg := range r.items {
		if !filter.IncludeDeleted && sg.IsDeleted() {
			continue
		}
		result = append(result, sg)
	}
	return result, nil
}

func (r *fakeServiceGroupRepository) GetByID(ctx context.Context, id string) (*domain.ServiceGroup, error) {
	sg, ok := r.items[id]
	if !ok || sg.IsDeleted() {
		return nil, nil
	}
	return sg, nil
}

func (r *fakeServiceGroupRepository) GetBySlug(ctx context.Context, slug string) (*domain.ServiceGroup, error) {
	for _, sg := range r.items {
		if sg.Slug() == slug && !sg.IsDeleted() {
			return sg, nil
		}
	}
	return nil, nil
}

func (r *fakeServiceGroupRepository) Create(ctx context.Context, serviceGroup *domain.ServiceGroup) (*domain.ServiceGroup, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateServiceGroup(
		id, serviceGroup.Name(), serviceGroup.Slug(),
		serviceGroup.ImageURL(), serviceGroup.MetaTitle(), serviceGroup.MetaDescription(),
		serviceGroup.IsFeatured(), serviceGroup.OrderIndex(), serviceGroup.CategoryIDs(),
		now, now, nil,
	)
	r.items[id] = created
	return created, nil
}

func (r *fakeServiceGroupRepository) Update(ctx context.Context, serviceGroup *domain.ServiceGroup) (*domain.ServiceGroup, error) {
	r.items[serviceGroup.ID()] = serviceGroup
	return serviceGroup, nil
}

func (r *fakeServiceGroupRepository) SoftDelete(ctx context.Context, id string) error {
	sg, ok := r.items[id]
	if !ok {
		return nil
	}
	sg.MarkDeleted(time.Now())
	return nil
}

func (r *fakeServiceGroupRepository) Restore(ctx context.Context, id string) error {
	sg, ok := r.items[id]
	if !ok {
		return nil
	}
	sg.Restore()
	return nil
}
