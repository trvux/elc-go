package application

import (
	"context"
	"sync"

	"github.com/trvux/elc-go/internal/system-page/domain"
)

type fakeSystemPageRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.SystemPage
}

func newFakeSystemPageRepository() *fakeSystemPageRepository {
	return &fakeSystemPageRepository{items: make(map[string]*domain.SystemPage)}
}

func (r *fakeSystemPageRepository) seed(p *domain.SystemPage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[p.ID()] = p
}

func (r *fakeSystemPageRepository) GetAll(ctx context.Context) ([]*domain.SystemPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.SystemPage, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, item)
	}
	return result, nil
}

func (r *fakeSystemPageRepository) GetByID(ctx context.Context, id string) (*domain.SystemPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if p, ok := r.items[id]; ok {
		return p, nil
	}
	return nil, nil
}

func (r *fakeSystemPageRepository) GetBySlug(ctx context.Context, slug string) (*domain.SystemPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, item := range r.items {
		if item.Slug() == slug {
			return item, nil
		}
	}
	return nil, nil
}

func (r *fakeSystemPageRepository) Update(ctx context.Context, p *domain.SystemPage) (*domain.SystemPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items[p.ID()] = p
	return p, nil
}
