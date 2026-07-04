//go:build integration

package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/trvux/elc-go/internal/group/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/group/infrastructure/...
func TestPostgresGroupRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresGroupRepository(pool)

	content := json.RawMessage(`{"type":"doc","content":[]}`)
	faq := []domain.FAQItem{{Question: "Bao hanh may nam?", Answer: "2 nam"}}
	imageUrl := "https://example.com/image.png"

	g, err := domain.NewGroup(
		"Integration Test Group", "integration-test-group-xyz", &imageUrl,
		nil, nil, false, 999, content, faq,
	)
	if err != nil {
		t.Fatalf("NewGroup failed: %v", err)
	}

	created, err := repo.Create(ctx, g)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM group_categories WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created group to have an ID")
	}
	if len(created.FAQ()) != 1 || created.FAQ()[0].Answer != "2 nam" {
		t.Errorf("expected faq to round-trip, got %+v", created.FAQ())
	}
	if created.Content() == nil {
		t.Error("expected content to round-trip")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Slug() != "integration-test-group-xyz" {
		t.Errorf("expected fetched group to match created one, got %+v", fetched)
	}

	bySlug, err := repo.GetBySlug(ctx, "integration-test-group-xyz")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if bySlug == nil || bySlug.ID() != created.ID() {
		t.Errorf("expected GetBySlug to find the created group, got %+v", bySlug)
	}

	if err := fetched.UpdateName("Integration Test Group Updated"); err != nil {
		t.Fatalf("UpdateName failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name() != "Integration Test Group Updated" {
		t.Errorf("expected updated name, got %s", updated.Name())
	}

	// Soft delete then verify it's excluded from GetByID and GetBySlug.
	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted group to not be found by GetByID")
	}
	if found, _ := repo.GetBySlug(ctx, "integration-test-group-xyz"); found != nil {
		t.Error("expected soft-deleted group to not be found by GetBySlug")
	}

	// Resurrect-on-create logic: creating a new group with the same slug should resurrect the old one.
	resInput, err := domain.NewGroup(
		"Resurrected Group", "integration-test-group-xyz", &imageUrl,
		nil, nil, true, 100, nil, nil,
	)
	if err != nil {
		t.Fatalf("NewGroup (resurrect) failed: %v", err)
	}
	resurrected, err := repo.Create(ctx, resInput)
	if err != nil {
		t.Fatalf("Create (resurrect) failed: %v", err)
	}
	if resurrected.ID() != created.ID() {
		t.Errorf("expected resurrect to reuse ID %s, got %s", created.ID(), resurrected.ID())
	}
	if resurrected.Name() != "Resurrected Group" {
		t.Errorf("expected resurrect to update fields, got name %s", resurrected.Name())
	}
	if resurrected.IsDeleted() {
		t.Error("expected resurrected group to not be deleted")
	}
}

func TestPostgresGroupRepository_SoftDeleteCascade(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresGroupRepository(pool)

	// 1. Create group category
	g, _ := domain.NewGroup("Cascade Test Group", "cascade-test-group-xyz", nil, nil, nil, false, 0, nil, nil)
	group, err := repo.Create(ctx, g)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM group_categories WHERE id = $1", group.ID())
	}()

	// 2. Fetch an existing project type
	var projectTypeID string
	err = pool.QueryRow(ctx, "SELECT id FROM project_type LIMIT 1").Scan(&projectTypeID)
	if err != nil {
		t.Fatalf("failed to fetch project type: %v", err)
	}

	// 3. Create categories belonging to the group
	var categoryID string
	err = pool.QueryRow(ctx, `
		INSERT INTO categories (name, slug, group_id, is_featured, order_index)
		VALUES ('Cascade Test Category', 'cascade-test-cat-xyz', $1, false, 0)
		RETURNING id`,
		group.ID(),
	).Scan(&categoryID)
	if err != nil {
		t.Fatalf("failed to insert category: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM categories WHERE id = $1", categoryID)
	}()

	// 4. Create project type category association
	_, err = pool.Exec(ctx, `
		INSERT INTO project_type_category (project_type_id, category_id)
		VALUES ($1, $2)`,
		projectTypeID, categoryID,
	)
	if err != nil {
		t.Fatalf("failed to insert project_type_category: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM project_type_category WHERE category_id = $1", categoryID)
	}()

	// Perform Soft Delete
	if err := repo.SoftDelete(ctx, group.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Verify Effect 1: group category soft-deleted
	var groupDeletedAt *time.Time
	err = pool.QueryRow(ctx, "SELECT deleted_at FROM group_categories WHERE id = $1", group.ID()).Scan(&groupDeletedAt)
	if err != nil {
		t.Fatalf("failed to query group: %v", err)
	}
	if groupDeletedAt == nil {
		t.Error("expected group deleted_at to be set")
	}

	// Verify Effect 2: category soft-deleted
	var categoryDeletedAt *time.Time
	err = pool.QueryRow(ctx, "SELECT deleted_at FROM categories WHERE id = $1", categoryID).Scan(&categoryDeletedAt)
	if err != nil {
		t.Fatalf("failed to query category: %v", err)
	}
	if categoryDeletedAt == nil {
		t.Error("expected category deleted_at to be set")
	}

	// Verify Effect 3: association row hard-deleted from project_type_category
	var count int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM project_type_category WHERE category_id = $1", categoryID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query project_type_category count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected project_type_category row to be deleted, got count %d", count)
	}
}
