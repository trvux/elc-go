package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/trvux/elc-go/internal/redirect/domain"
)

type fakeRepository struct {
	items map[string]*domain.Redirect // by id
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{items: map[string]*domain.Redirect{}}
}

func (r *fakeRepository) seed(from, to string) {
	d, err := domain.NewRedirect(from, to, "", domain.SourceImport)
	if err != nil {
		panic(err)
	}
	if _, err := r.Create(context.Background(), d); err != nil {
		panic(err)
	}
}

func (r *fakeRepository) targetOf(from string) string {
	for _, it := range r.items {
		if it.FromPath() == from {
			return it.ToPath()
		}
	}
	return ""
}

func (r *fakeRepository) List(ctx context.Context, f domain.Filter) ([]*domain.Redirect, int, error) {
	out := []*domain.Redirect{}
	for _, it := range r.items {
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FromPath() < out[j].FromPath() })
	return out, len(out), nil
}

func (r *fakeRepository) GetByID(ctx context.Context, id string) (*domain.Redirect, error) {
	return r.items[id], nil
}

func (r *fakeRepository) GetByFrom(ctx context.Context, from string) (*domain.Redirect, error) {
	for _, it := range r.items {
		if it.FromPath() == from {
			return it, nil
		}
	}
	return nil, nil
}

func (r *fakeRepository) Create(ctx context.Context, d *domain.Redirect) (*domain.Redirect, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateRedirect(id, d.FromPath(), d.ToPath(), d.IsActive(), d.Source(), d.Note(), now, now)
	r.items[id] = created
	return created, nil
}

func (r *fakeRepository) Update(ctx context.Context, d *domain.Redirect) (*domain.Redirect, error) {
	r.items[d.ID()] = d
	return d, nil
}

func (r *fakeRepository) Delete(ctx context.Context, id string) error {
	delete(r.items, id)
	return nil
}

func (r *fakeRepository) Repoint(ctx context.Context, oldTarget, newTarget string) error {
	for id, it := range r.items {
		if it.ToPath() == oldTarget && it.FromPath() != newTarget {
			r.items[id] = domain.RehydrateRedirect(it.ID(), it.FromPath(), newTarget, it.IsActive(), it.Source(), it.Note(), it.CreatedAt(), time.Now())
		}
	}
	return nil
}
