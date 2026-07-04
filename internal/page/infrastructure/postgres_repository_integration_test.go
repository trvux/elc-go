//go:build integration

package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/trvux/elc-go/internal/page/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

func TestPostgresPageRepository_CRUD(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresPageRepository(pool)

	content := json.RawMessage(`{"text": "test"}`)
	p, err := domain.NewPage(
		"ELC Page Integration", "elc-page-integration-xyz", content,
		true, nil, nil, 999,
	)
	if err != nil {
		t.Fatalf("NewPage failed: %v", err)
	}

	created, err := repo.Create(ctx, p)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		// Hard delete in database to clean up integration test
		_, _ = pool.Exec(ctx, "DELETE FROM pages WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created page to have an ID")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Slug() != "elc-page-integration-xyz" {
		t.Errorf("expected fetched page to match created one, got %+v", fetched)
	}

	bySlug, err := repo.GetBySlug(ctx, "elc-page-integration-xyz")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if bySlug == nil || bySlug.ID() != created.ID() {
		t.Errorf("expected GetBySlug to find the created page, got %+v", bySlug)
	}

	// Update page
	if err := fetched.Update("ELC Page Integration Updated", "elc-page-integration-xyz", content, true, nil, nil, 999); err != nil {
		t.Fatalf("Update entity failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title() != "ELC Page Integration Updated" {
		t.Errorf("expected updated title, got %s", updated.Title())
	}

	// Soft delete
	if err := repo.Delete(ctx, created.ID()); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify not found by ID anymore
	found, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID post-delete failed: %v", err)
	}
	if found != nil {
		t.Error("expected soft-deleted page to not be found by GetByID")
	}

	// Verify count does not include deleted
	count, err := repo.Count(ctx, domain.PageFilter{Search: "ELC Page Integration"})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count to be 0, got %d", count)
	}

	// Restore
	if err := repo.Restore(ctx, created.ID()); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	found, err = repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID post-restore failed: %v", err)
	}
	if found == nil {
		t.Error("expected restored page to be found again")
	}
}
