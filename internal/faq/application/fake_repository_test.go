package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/trvux/elc-go/internal/faq/domain"
)

type fakeFAQRepository struct {
	items map[string]*domain.FAQ
}

func newFakeFAQRepository() *fakeFAQRepository {
	return &fakeFAQRepository{items: map[string]*domain.FAQ{}}
}

func (r *fakeFAQRepository) GetByOwner(ctx context.Context, filter domain.FAQFilter) ([]*domain.FAQ, error) {
	result := []*domain.FAQ{}
	for _, f := range r.items {
		if f.OwnerType() != filter.OwnerType || f.OwnerID() != filter.OwnerID {
			continue
		}
		if filter.PublishedOnly && !f.IsPublished() {
			continue
		}
		result = append(result, f)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].OrderIndex() < result[j].OrderIndex() })
	return result, nil
}

func (r *fakeFAQRepository) GetByID(ctx context.Context, id string) (*domain.FAQ, error) {
	f, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	return f, nil
}

func (r *fakeFAQRepository) Create(ctx context.Context, faq *domain.FAQ) (*domain.FAQ, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateFAQ(
		id, faq.OwnerType(), faq.OwnerID(), faq.Question(), faq.Answer(),
		faq.OrderIndex(), faq.IsPublished(), now, now,
	)
	r.items[id] = created
	return created, nil
}

func (r *fakeFAQRepository) Update(ctx context.Context, faq *domain.FAQ) (*domain.FAQ, error) {
	r.items[faq.ID()] = faq
	return faq, nil
}

func (r *fakeFAQRepository) Delete(ctx context.Context, id string) error {
	delete(r.items, id)
	return nil
}
