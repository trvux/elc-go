package application

import (
	"context"
	"fmt"

	"github.com/trvux/elc-go/internal/event/domain"
)

type fakeEventRepository struct {
	events map[string]*domain.Event
}

func newFakeEventRepository() *fakeEventRepository {
	return &fakeEventRepository{events: map[string]*domain.Event{}}
}

func (r *fakeEventRepository) Create(ctx context.Context, event *domain.Event) error {
	id := fmt.Sprintf("id-%d", len(r.events)+1)
	r.events[id] = domain.RehydrateEvent(
		id, event.Name(), event.EntityType(), event.EntityID(), event.PagePath(), event.SessionID(), event.CreatedAt(),
	)
	return nil
}

func (r *fakeEventRepository) TopViewed(ctx context.Context, filter domain.TopViewedFilter) ([]domain.EntityViewCount, error) {
	counts := map[string]int{}
	for _, e := range r.events {
		if e.Name() != domain.EventViewItem {
			continue
		}
		if e.EntityType() == nil || *e.EntityType() != filter.EntityType {
			continue
		}
		if e.EntityID() == nil {
			continue
		}
		counts[*e.EntityID()]++
	}
	result := make([]domain.EntityViewCount, 0, len(counts))
	for id, count := range counts {
		result = append(result, domain.EntityViewCount{EntityID: id, Count: count})
	}
	return result, nil
}
