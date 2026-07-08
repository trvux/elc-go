//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/project/domain"
)

// Run explicitly with: go test -tags=integration ./internal/project/infrastructure/...
func TestPostgresProjectRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var categoryID string
	if err := pool.QueryRow(ctx, "SELECT id FROM categories WHERE deleted_at IS NULL LIMIT 1").Scan(&categoryID); err != nil {
		t.Skipf("no category row available: %v", err)
	}

	repo := NewPostgresProjectRepository(pool)

	p, err := domain.NewProject(
		"Integration Test Project", "integration-test-project-xyz",
		nil, nil, false, true, nil, nil, domain.Seo{}, 999, nil,
		"", "", nil, "", "",
	)
	if err != nil {
		t.Fatalf("NewProject failed: %v", err)
	}

	created, err := repo.Create(ctx, p, []domain.CategoryCondition{{CategoryID: categoryID, Condition: "new"}}, nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM project_category WHERE project_id = $1", created.ID())
		_, _ = pool.Exec(ctx, "DELETE FROM projects WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created project to have an ID")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Slug() != "integration-test-project-xyz" {
		t.Fatalf("expected fetched project to match created one, got %+v", fetched)
	}
	if len(fetched.Categories) != 1 || fetched.Categories[0].ID != categoryID || fetched.Categories[0].Condition != "new" {
		t.Errorf("expected 1 category %s/new, got %+v", categoryID, fetched.Categories)
	}

	bySlug, err := repo.GetBySlug(ctx, "integration-test-project-xyz", false)
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if bySlug == nil || bySlug.ID() != created.ID() {
		t.Errorf("expected GetBySlug to find the created project, got %+v", bySlug)
	}

	// Update: omit relations (nil) — must leave the existing category/
	// service relations untouched, mirroring the old TS
	// `input.categories !== undefined` distinction. See domain/types.go's
	// UpdateProjectInput doc comment.
	if err := fetched.UpdateTitle("Integration Test Project Updated"); err != nil {
		t.Fatalf("UpdateTitle failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched.Project, nil, nil, nil)
	if err != nil {
		t.Fatalf("Update (relations omitted) failed: %v", err)
	}
	if updated.Title() != "Integration Test Project Updated" {
		t.Errorf("expected updated title, got %s", updated.Title())
	}
	afterOmit, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID after omitted-relations update failed: %v", err)
	}
	if len(afterOmit.Categories) != 1 {
		t.Errorf("expected category relation to remain untouched, got %+v", afterOmit.Categories)
	}

	// Update: explicitly empty relations — must clear them.
	emptyCategories := []domain.CategoryCondition{}
	updated, err = repo.Update(ctx, updated, &emptyCategories, nil, nil)
	if err != nil {
		t.Fatalf("Update (relations cleared) failed: %v", err)
	}
	afterClear, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID after cleared-relations update failed: %v", err)
	}
	if len(afterClear.Categories) != 0 {
		t.Errorf("expected categories cleared, got %+v", afterClear.Categories)
	}
	_ = updated

	// Soft delete then verify it's excluded from GetByID/GetBySlug.
	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted project to not be found by GetByID")
	}
	if found, _ := repo.GetBySlug(ctx, "integration-test-project-xyz", false); found != nil {
		t.Error("expected soft-deleted project to not be found by GetBySlug")
	}

	if err := repo.Restore(ctx, created.ID()); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found == nil {
		t.Error("expected restored project to be found again")
	}
}

// TestPostgresProjectRepository_ResurrectOnSlugReuse exercises the mandatory
// (not business-optional) resurrect: projects.slug is a plain UNIQUE
// constraint (unlike brand/group/category's partial index), so re-creating
// with the same slug while the old row is soft-deleted MUST reuse the same
// row/ID — a plain INSERT would otherwise violate the constraint outright.
// See docs/project.md.
func TestPostgresProjectRepository_ResurrectOnSlugReuse(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var categoryID string
	if err := pool.QueryRow(ctx, "SELECT id FROM categories WHERE deleted_at IS NULL LIMIT 1").Scan(&categoryID); err != nil {
		t.Skipf("no category row available: %v", err)
	}

	repo := NewPostgresProjectRepository(pool)

	p1, _ := domain.NewProject("Resurrect Test", "integration-test-resurrect-xyz", nil, nil, false, true, nil, nil, domain.Seo{}, 0, nil, "", "", nil, "", "")
	created1, err := repo.Create(ctx, p1, nil, nil, nil)
	if err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM projects WHERE id = $1", created1.ID())
	}()

	if err := repo.SoftDelete(ctx, created1.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	p2, _ := domain.NewProject("Resurrect Test Again", "integration-test-resurrect-xyz", nil, nil, false, true, nil, nil, domain.Seo{}, 0, nil, "", "", nil, "", "")
	created2, err := repo.Create(ctx, p2, nil, nil, nil)
	if err != nil {
		t.Fatalf("second Create (resurrect) failed: %v", err)
	}

	if created2.ID() != created1.ID() {
		t.Errorf("expected resurrect to reuse id %s, got new id %s", created1.ID(), created2.ID())
	}
	if created2.Title() != "Resurrect Test Again" {
		t.Errorf("expected resurrected row's title updated, got %s", created2.Title())
	}
	if created2.IsDeleted() {
		t.Error("expected resurrected row to no longer be soft-deleted")
	}
}

// TestPostgresProjectRepository_CreateRollsBackOnBadRelation verifies Create
// is atomic: an invalid category id in the relations payload must roll back
// the projects row insert too, not leave an orphaned project with no
// matching project_category rows. The old TS create() ran these as separate
// un-transactioned Supabase calls, so a failure here used to leave exactly
// that kind of half-applied state — see docs/project.md.
func TestPostgresProjectRepository_CreateRollsBackOnBadRelation(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var categoryID string
	if err := pool.QueryRow(ctx, "SELECT id FROM categories WHERE deleted_at IS NULL LIMIT 1").Scan(&categoryID); err != nil {
		t.Skipf("no category row available: %v", err)
	}

	repo := NewPostgresProjectRepository(pool)

	p, _ := domain.NewProject("Rollback Test", "integration-test-rollback-xyz", nil, nil, false, true, nil, nil, domain.Seo{}, 0, nil, "", "", nil, "", "")
	_, err = repo.Create(ctx, p, []domain.CategoryCondition{{CategoryID: "00000000-0000-0000-0000-000000000000", Condition: "new"}}, nil, nil)
	if err == nil {
		t.Fatal("expected Create to fail on a nonexistent category id")
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM projects WHERE slug = $1", "integration-test-rollback-xyz").Scan(&count); err != nil {
		t.Fatalf("failed to check for leftover row: %v", err)
	}
	if count != 0 {
		t.Errorf("expected failed Create to leave no projects row behind, found %d", count)
		_, _ = pool.Exec(ctx, "DELETE FROM projects WHERE slug = $1", "integration-test-rollback-xyz")
	}
}

func TestPostgresProjectRepository_GetAdjacent(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var projectID string
	if err := pool.QueryRow(ctx,
		"SELECT id FROM projects WHERE deleted_at IS NULL AND is_published = true LIMIT 1",
	).Scan(&projectID); err != nil {
		t.Skipf("no published project row available to test against: %v", err)
	}

	repo := NewPostgresProjectRepository(pool)

	prev, next, err := repo.GetAdjacent(ctx, nil, projectID)
	if err != nil {
		t.Fatalf("GetAdjacent failed: %v", err)
	}
	if prev == nil && next == nil {
		t.Log("no adjacent project found — acceptable only if this is the sole published project")
	}
}

func TestPostgresProjectRepository_GetBySlugWithPricing(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var slug string
	if err := pool.QueryRow(ctx,
		"SELECT slug FROM projects WHERE deleted_at IS NULL LIMIT 1",
	).Scan(&slug); err != nil {
		t.Skipf("no project row available: %v", err)
	}

	repo := NewPostgresProjectRepository(pool)

	withoutPricing, err := repo.GetBySlug(ctx, slug, false)
	if err != nil {
		t.Fatalf("GetBySlug(withPricing=false) failed: %v", err)
	}
	for _, c := range withoutPricing.Categories {
		if c.LowPrice != 0 || c.HighPrice != 0 || c.OfferCount != 0 {
			t.Errorf("expected zero pricing when withPricing=false, got %+v", c)
		}
	}

	// Just verifying the pricing-enabled query runs without error against
	// real data — actual price values depend on live product data.
	if _, err := repo.GetBySlug(ctx, slug, true); err != nil {
		t.Fatalf("GetBySlug(withPricing=true) failed: %v", err)
	}
}
