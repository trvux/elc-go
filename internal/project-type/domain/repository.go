package domain

import "context"

// ProjectTypeRepository — GetAll/GetByID return *ProjectTypeWithCategories
// (joined with project_type_category + categories + group_categories) so the
// HTTP response can carry the nested categories without extra round trips.
// Create/Update take a plain *ProjectType plus the join-table payload
// (category ids) as a separate parameter, and are responsible for writing
// both atomically — the old TS repository ran the equivalent as several
// sequential, un-transactioned Supabase calls (find-existing, insert/update,
// delete old relations, insert new relations), see
// internal/project-type/infrastructure/postgres_repository.go.
type ProjectTypeRepository interface {
	GetAll(ctx context.Context, filter ProjectTypeFilter) ([]*ProjectTypeWithCategories, error)
	Count(ctx context.Context, filter ProjectTypeFilter) (int, error)
	GetByID(ctx context.Context, id string) (*ProjectTypeWithCategories, error)
	Create(ctx context.Context, projectType *ProjectType, categoryIDs []string) (*ProjectType, error)
	Update(ctx context.Context, projectType *ProjectType, categoryIDs *[]string) (*ProjectType, error)
	// SoftDelete mirrors the old TS delete() exactly: soft-delete the
	// project_type row, null out project_type_id on referencing projects
	// (the FK's own ON DELETE SET NULL never fires because this is an
	// UPDATE, not a real DELETE), and hard-delete this project type's
	// project_type_category rows — all three in one transaction, fixing the
	// old TS's un-transactioned three-call version. See
	// modules/project-type/infrastructure/projectTypeRepo.ts's delete().
	SoftDelete(ctx context.Context, id string) error
}
