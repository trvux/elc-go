package application

import (
	"context"

	"github.com/trvux/elc-go/internal/project/domain"
)

// DeleteProject soft-deletes only the projects row — the old TS delete()
// never touched project_category/project_service either, leaving those join
// rows pointing at a soft-deleted project (same "leftover FK reference"
// precedent as brand's products.brand_id, see docs/brand.md). See
// docs/project.md.
func DeleteProject(ctx context.Context, repo domain.ProjectRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreProject(ctx context.Context, repo domain.ProjectRepository, id string) error {
	return repo.Restore(ctx, id)
}
