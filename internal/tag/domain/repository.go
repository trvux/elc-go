package domain

import "context"

type TagRepository interface {
	GetAll(ctx context.Context, filter TagFilter) ([]*Tag, error)
	GetByID(ctx context.Context, id string) (*Tag, error)
	GetBySlug(ctx context.Context, slug string) (*Tag, error)
	// GetByIDs batch-loads tags for a set of ids — used by other modules'
	// repositories (news/catalog/project) to resolve TagRefs without an N+1.
	GetByIDs(ctx context.Context, ids []string) ([]*Tag, error)
	Create(ctx context.Context, tag *Tag) (*Tag, error)
	Update(ctx context.Context, tag *Tag) (*Tag, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
