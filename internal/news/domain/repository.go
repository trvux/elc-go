package domain

import "context"

type NewsRepository interface {
	GetAll(ctx context.Context, filter NewsFilter) ([]*News, error)
	Count(ctx context.Context, filter NewsFilter) (int, error)
	GetByID(ctx context.Context, id string) (*News, error)
	GetBySlug(ctx context.Context, slug string) (*News, error)
	// Create resurrects a soft-deleted row sharing the same slug rather than
	// plain-INSERTing — news.slug is a plain UNIQUE constraint (confirmed via
	// `\d news`, unlike brand/group/category's partial "unique among
	// non-deleted" index), so a plain INSERT would fail the constraint
	// outright on slug reuse. See docs/news.md.
	Create(ctx context.Context, news *News) (*News, error)
	Update(ctx context.Context, news *News) (*News, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
