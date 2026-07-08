package domain

import "context"

// ProjectRepository — GetAll/GetByID/GetBySlug return *ProjectWithRelations
// (joined with project_type/categories/services) so the HTTP response can
// carry nested objects without extra round trips; Create/Update take a plain
// *Project plus the join-table payloads (categories with condition, service
// ids) as separate parameters, and are responsible for writing all of it
// atomically — see internal/project/infrastructure/postgres_repository.go
// for why this needs a transaction (the old TS ran the equivalent writes as
// several sequential, un-transactioned Supabase calls).
type ProjectRepository interface {
	GetAll(ctx context.Context, filter ProjectFilter) ([]*ProjectWithRelations, error)
	Count(ctx context.Context, filter ProjectFilter) (int, error)
	GetByID(ctx context.Context, id string) (*ProjectWithRelations, error)
	// GetBySlug's withPricing flag mirrors the old TS split: false is the
	// old projectRepo.ts's getBySlug shape (categories without pricing,
	// cheap), true is the old infrastructure/resolveProjectPath.ts's shape
	// (extra JOIN into products for lowPrice/highPrice/offerCount, used only
	// by the public project detail page). See docs/project.md.
	GetBySlug(ctx context.Context, slug string, withPricing bool) (*ProjectWithRelations, error)
	Create(ctx context.Context, project *Project, categories []CategoryCondition, serviceIDs []string, tagIDs []string) (*Project, error)
	Update(ctx context.Context, project *Project, categories *[]CategoryCondition, serviceIDs *[]string, tagIDs *[]string) (*Project, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	UpdateOrder(ctx context.Context, id string, orderIndex int) error
	TogglePublish(ctx context.Context, id string, isPublished bool) error
	ToggleFeatured(ctx context.Context, id string, isFeatured bool) error
	// GetAdjacent resolves prev/next within projectTypeID first, falling
	// back to the full published catalog of projects when that project type
	// has fewer than 2 published siblings — same "same-group siblings
	// first, fall back to everything published" rule as
	// internal/catalog/domain/repository.go's GetAdjacent, pushed down to
	// SQL instead of the old TS application layer loading everything into
	// memory to sort (see modules/project/application/getAdjacentProjects.ts).
	GetAdjacent(ctx context.Context, projectTypeID *string, currentID string) (prev, next *AdjacentProject, err error)
	GetCategoriesByProjectTypeID(ctx context.Context, projectTypeID string) ([]CategoryRef, error)
}
