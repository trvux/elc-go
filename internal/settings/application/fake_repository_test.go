package application

import (
	"context"
	"sync"

	"github.com/trvux/elc-go/internal/settings/domain"
)

type fakeSettingsRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.SiteSetting
}

func newFakeSettingsRepository() *fakeSettingsRepository {
	return &fakeSettingsRepository{
		items: make(map[string]*domain.SiteSetting),
	}
}

func (r *fakeSettingsRepository) GetAll(ctx context.Context) ([]*domain.SiteSetting, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.SiteSetting, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, item)
	}
	return result, nil
}

func (r *fakeSettingsRepository) UpdateMany(ctx context.Context, settings []*domain.SiteSetting) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range settings {
		r.items[s.Key()] = s
	}
	return nil
}
