package application

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/trvux/elc-go/internal/page/domain"
)

type fakePageRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.Page
}

func newFakePageRepository() *fakePageRepository {
	return &fakePageRepository{
		items: make(map[string]*domain.Page),
	}
}

func (r *fakePageRepository) GetAll(ctx context.Context, filter domain.PageFilter) ([]*domain.Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Page
	for _, p := range r.items {
		if !filter.IncludeDeleted && p.DeletedAt() != nil {
			continue
		}
		if filter.IsPublished != nil && p.IsPublished() != *filter.IsPublished {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(p.Title()), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, p)
	}

	return result, nil
}

func (r *fakePageRepository) Count(ctx context.Context, filter domain.PageFilter) (int, error) {
	list, err := r.GetAll(ctx, filter)
	if err != nil {
		return 0, err
	}
	return len(list), nil
}

func (r *fakePageRepository) GetByID(ctx context.Context, id string) (*domain.Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.items[id]
	if !ok || p.DeletedAt() != nil {
		return nil, nil
	}
	return p, nil
}

func (r *fakePageRepository) GetBySlug(ctx context.Context, slug string) (*domain.Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.items {
		if p.Slug() == slug && p.DeletedAt() == nil {
			return p, nil
		}
	}
	return nil, nil
}

func (r *fakePageRepository) Create(ctx context.Context, page *domain.Page) (*domain.Page, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := uuid.NewString()
	created := domain.RehydratePage(
		id,
		page.Title(),
		page.Slug(),
		page.Content(),
		page.IsPublished(),
		page.MetaTitle(),
		page.MetaDescription(),
		page.OrderIndex(),
		time.Now(),
		time.Now(),
		nil,
	)
	r.items[id] = created
	return created, nil
}

func (r *fakePageRepository) Update(ctx context.Context, page *domain.Page) (*domain.Page, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items[page.ID()] = page
	return page, nil
}

func (r *fakePageRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.items[id]
	if ok {
		now := time.Now()
		deleted := domain.RehydratePage(
			p.ID(),
			p.Title(),
			p.Slug(),
			p.Content(),
			p.IsPublished(),
			p.MetaTitle(),
			p.MetaDescription(),
			p.OrderIndex(),
			p.CreatedAt(),
			p.UpdatedAt(),
			&now,
		)
		r.items[id] = deleted
	}
	return nil
}

func (r *fakePageRepository) Restore(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.items[id]
	if ok {
		restored := domain.RehydratePage(
			p.ID(),
			p.Title(),
			p.Slug(),
			p.Content(),
			p.IsPublished(),
			p.MetaTitle(),
			p.MetaDescription(),
			p.OrderIndex(),
			p.CreatedAt(),
			p.UpdatedAt(),
			nil,
		)
		r.items[id] = restored
	}
	return nil
}
