package domain

import "context"

// CategoryRepository — GetAll/GetByID/GetBySlug return *CategoryWithRelations
// (joined with group_categories) so the HTTP response can carry the nested
// group object without a second round trip; Create/Update return a plain
// *Category — same split catalog/service already use for their own
// relations, see internal/catalog/domain/repository.go.
type CategoryRepository interface {
	GetAll(ctx context.Context, filter CategoryFilter) ([]*CategoryWithRelations, error)
	Count(ctx context.Context, filter CategoryFilter) (int, error)
	GetByID(ctx context.Context, id string) (*CategoryWithRelations, error)
	GetBySlug(ctx context.Context, slug string) (*CategoryWithRelations, error)
	Create(ctx context.Context, category *Category) (*Category, error)
	Update(ctx context.Context, category *Category) (*Category, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
