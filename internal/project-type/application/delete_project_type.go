package application

import (
	"context"

	"github.com/trvux/elc-go/internal/project-type/domain"
)

// DeleteProjectType soft-deletes the project type, nulls out project_type_id
// on referencing projects, and clears project_type_category rows — all three
// atomically inside repo.SoftDelete. See domain.ProjectTypeRepository's doc
// comment for why (the old TS delete() ran these as three separate,
// un-transactioned Supabase calls).
func DeleteProjectType(ctx context.Context, repo domain.ProjectTypeRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}
