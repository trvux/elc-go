//go:build integration

package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/category/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/category/infrastructure/...
func TestPostgresCategoryRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresCategoryRepository(pool)

	content := json.RawMessage(`{"type":"doc","content":[]}`)
	imageUrl := "https://example.com/image.png"

	c, err := domain.NewCategory(
		"Integration Test Category", "integration-test-category-xyz", nil, &imageUrl,
		nil, nil, false, 999, content,
	)
	if err != nil {
		t.Fatalf("NewCategory failed: %v", err)
	}

	created, err := repo.Create(ctx, c)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM categories WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created category to have an ID")
	}
	if created.Content() == nil {
		t.Error("expected content to round-trip")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Slug() != "integration-test-category-xyz" {
		t.Errorf("expected fetched category to match created one, got %+v", fetched)
	}
	if fetched.Group != nil {
		t.Errorf("expected no group ref for a category with nil group_id, got %+v", fetched.Group)
	}

	bySlug, err := repo.GetBySlug(ctx, "integration-test-category-xyz")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if bySlug == nil || bySlug.ID() != created.ID() {
		t.Errorf("expected GetBySlug to find the created category, got %+v", bySlug)
	}

	if err := fetched.UpdateName("Integration Test Category Updated"); err != nil {
		t.Fatalf("UpdateName failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched.Category)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name() != "Integration Test Category Updated" {
		t.Errorf("expected updated name, got %s", updated.Name())
	}

	// SoftDelete must succeed even when there are no project_type_category
	// rows referencing this category — the common case.
	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted category to not be found by GetByID")
	}
	if found, _ := repo.GetBySlug(ctx, "integration-test-category-xyz"); found != nil {
		t.Error("expected soft-deleted category to not be found by GetBySlug")
	}

	if err := repo.Restore(ctx, created.ID()); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found == nil {
		t.Error("expected restored category to be found again")
	}

	// Resurrect-on-create: soft-delete again, then create with the same slug.
	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete (for resurrect) failed: %v", err)
	}
	resurrectInput, err := domain.NewCategory(
		"Resurrected Category", "integration-test-category-xyz", nil, nil,
		nil, nil, false, 0, nil,
	)
	if err != nil {
		t.Fatalf("NewCategory (resurrect) failed: %v", err)
	}
	resurrected, err := repo.Create(ctx, resurrectInput)
	if err != nil {
		t.Fatalf("Create (resurrect) failed: %v", err)
	}
	if resurrected.ID() != created.ID() {
		t.Errorf("expected resurrect to reuse ID %s, got %s", created.ID(), resurrected.ID())
	}
	if resurrected.IsDeleted() {
		t.Error("expected resurrected category to not be deleted")
	}
}
