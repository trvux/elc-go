package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/news/domain"
)

// fakeNewsRepository is an in-memory stand-in used only to exercise
// application-layer orchestration (validation, partial update). Real SQL
// behavior (resurrect-on-create, ordering) is covered by the integration
// tests against the real DB in internal/news/infrastructure.
type fakeNewsRepository struct {
	items map[string]*domain.News
	tags  map[string][]string
}

func newFakeNewsRepository() *fakeNewsRepository {
	return &fakeNewsRepository{items: map[string]*domain.News{}, tags: map[string][]string{}}
}

func (r *fakeNewsRepository) GetAll(ctx context.Context, filter domain.NewsFilter) ([]*domain.News, error) {
	var result []*domain.News
	for _, n := range r.items {
		if !filter.IncludeDeleted && n.IsDeleted() {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(n.Title()), strings.ToLower(filter.Search)) {
			continue
		}
		if filter.CategoryID != nil && (n.CategoryID() == nil || *n.CategoryID() != *filter.CategoryID) {
			continue
		}
		if filter.ExcludeID != nil && n.ID() == *filter.ExcludeID {
			continue
		}
		result = append(result, n)
	}
	return result, nil
}

func (r *fakeNewsRepository) Count(ctx context.Context, filter domain.NewsFilter) (int, error) {
	all, _ := r.GetAll(ctx, filter)
	return len(all), nil
}

func (r *fakeNewsRepository) GetByID(ctx context.Context, id string) (*domain.News, error) {
	n, ok := r.items[id]
	if !ok || n.IsDeleted() {
		return nil, nil
	}
	return n, nil
}

func (r *fakeNewsRepository) GetBySlug(ctx context.Context, slug string) (*domain.News, error) {
	for _, n := range r.items {
		if n.Slug() == slug && !n.IsDeleted() {
			return n, nil
		}
	}
	return nil, nil
}

func (r *fakeNewsRepository) Create(ctx context.Context, news *domain.News, tagIDs []string) (*domain.News, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateNews(
		id, news.Title(), news.Slug(), news.Images(), news.Content(), news.Excerpt(), news.CategoryID(), news.AuthorID(),
		news.IsPublished(), news.MetaTitle(), news.MetaDescription(), news.Seo(), news.OrderIndex(),
		now, now, nil,
	)
	r.items[id] = created
	r.tags[id] = tagIDs
	return created, nil
}

func (r *fakeNewsRepository) Update(ctx context.Context, news *domain.News, tagIDs *[]string) (*domain.News, error) {
	r.items[news.ID()] = news
	if tagIDs != nil {
		r.tags[news.ID()] = *tagIDs
	}
	return news, nil
}

func (r *fakeNewsRepository) SoftDelete(ctx context.Context, id string) error {
	n, ok := r.items[id]
	if !ok {
		return nil
	}
	n.MarkDeleted(time.Now())
	return nil
}

func (r *fakeNewsRepository) Restore(ctx context.Context, id string) error {
	n, ok := r.items[id]
	if !ok {
		return nil
	}
	n.Restore()
	return nil
}
