//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/project-type/domain"
)

// Run explicitly with: go test -tags=integration ./internal/project-type/infrastructure/...
func TestPostgresProjectTypeRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresProjectTypeRepository(pool)

	var categoryID string
	if err := pool.QueryRow(ctx, "SELECT id FROM categories WHERE deleted_at IS NULL LIMIT 1").Scan(&categoryID); err != nil {
		t.Skipf("no category row available to attach: %v", err)
	}

	pt, err := domain.NewProjectType("Integration Test Project Type", "integration-test-project-type-xyz", nil, nil, nil, false, 999, nil)
	if err != nil {
		t.Fatalf("NewProjectType failed: %v", err)
	}

	created, err := repo.Create(ctx, pt, []string{categoryID})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM project_type_category WHERE project_type_id = $1", created.ID())
		_, _ = pool.Exec(ctx, "DELETE FROM project_type WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created project type to have an ID")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Slug() != "integration-test-project-type-xyz" {
		t.Errorf("expected fetched project type to match created one, got %+v", fetched)
	}
	if len(fetched.Categories) != 1 || fetched.Categories[0].ID != categoryID {
		t.Errorf("expected 1 attached category %s, got %+v", categoryID, fetched.Categories)
	}

	newName := "Integration Test Project Type Updated"
	if err := fetched.Update(domain.UpdateProjectTypeInput{Name: &newName}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	emptyCategories := []string{}
	updated, err := repo.Update(ctx, fetched.ProjectType, &emptyCategories)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name() != "Integration Test Project Type Updated" {
		t.Errorf("expected updated name, got %s", updated.Name())
	}

	afterUpdate, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if len(afterUpdate.Categories) != 0 {
		t.Errorf("expected categories cleared by explicit empty update, got %+v", afterUpdate.Categories)
	}

	// SoftDelete then verify it's excluded from GetByID and the join table
	// got cleaned up.
	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted project type to not be found by GetByID")
	}

	var joinCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM project_type_category WHERE project_type_id = $1", created.ID()).Scan(&joinCount); err != nil {
		t.Fatalf("failed to count project_type_category rows: %v", err)
	}
	if joinCount != 0 {
		t.Errorf("expected project_type_category rows cleared on soft delete, got %d", joinCount)
	}
}

// TestPostgresProjectTypeRepository_SoftDeleteNullsReferencingProjects
// exercises the delete()'s second step: projects.project_type_id must be
// nulled out for any project referencing the deleted project type, since the
// FK's ON DELETE SET NULL never fires on a soft delete (an UPDATE, not a
// real DELETE). See docs/project-type.md.
func TestPostgresProjectTypeRepository_SoftDeleteNullsReferencingProjects(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresProjectTypeRepository(pool)

	var categoryID string
	if err := pool.QueryRow(ctx, "SELECT id FROM categories WHERE deleted_at IS NULL LIMIT 1").Scan(&categoryID); err != nil {
		t.Skipf("no category row available: %v", err)
	}

	pt, err := domain.NewProjectType("FK Cleanup Test Project Type", "fk-cleanup-test-project-type-xyz", nil, nil, nil, false, 0, nil)
	if err != nil {
		t.Fatalf("NewProjectType failed: %v", err)
	}
	created, err := repo.Create(ctx, pt, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM project_type WHERE id = $1", created.ID())
	}()

	var projectID string
	err = pool.QueryRow(ctx, `
		INSERT INTO projects (category_id, title, slug, project_type_id)
		VALUES ($1, 'FK Cleanup Test Project', 'fk-cleanup-test-project-xyz', $2)
		RETURNING id`,
		categoryID, created.ID(),
	).Scan(&projectID)
	if err != nil {
		t.Fatalf("failed to insert test project: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM projects WHERE id = $1", projectID)
	}()

	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	var projectTypeID *string
	if err := pool.QueryRow(ctx, "SELECT project_type_id FROM projects WHERE id = $1", projectID).Scan(&projectTypeID); err != nil {
		t.Fatalf("failed to read back project: %v", err)
	}
	if projectTypeID != nil {
		t.Errorf("expected project.project_type_id to be nulled after project type soft delete, got %v", *projectTypeID)
	}
}
