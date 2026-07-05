//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/trvux/elc-go/internal/platform/db"
)

// system_pages rows are a fixed, admin-seeded set (no Create/Delete in this
// module), so this test reads an existing row, updates it, verifies, then
// restores its original meta fields rather than inserting/deleting a row.
func TestPostgresSystemPageRepository_ReadAndUpdate(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresSystemPageRepository(pool)

	list, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least one system page in the live DB")
	}

	target := list[0]
	for _, p := range list {
		if p.Slug() == "co-so-ha-tang" {
			target = p
			break
		}
	}

	byID, err := repo.GetByID(ctx, target.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if byID == nil || byID.ID() != target.ID() {
		t.Fatalf("GetByID mismatch: %+v", byID)
	}

	bySlug, err := repo.GetBySlug(ctx, target.Slug())
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if bySlug == nil || bySlug.ID() != target.ID() {
		t.Fatalf("GetBySlug mismatch: %+v", bySlug)
	}

	originalTitle := target.MetaTitle()
	originalDesc := target.MetaDescription()
	defer func() {
		if err := target.UpdateMeta(originalTitle, originalDesc); err != nil {
			t.Errorf("failed to restore original meta: %v", err)
			return
		}
		if _, err := repo.Update(ctx, target); err != nil {
			t.Errorf("failed to persist restored meta: %v", err)
		}
	}()

	testTitle := "integration test title"
	testDesc := "integration test description"
	if err := target.UpdateMeta(&testTitle, &testDesc); err != nil {
		t.Fatalf("unexpected UpdateMeta error: %v", err)
	}

	updated, err := repo.Update(ctx, target)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.MetaTitle() == nil || *updated.MetaTitle() != testTitle {
		t.Errorf("expected meta title %q, got %+v", testTitle, updated.MetaTitle())
	}
	if updated.MetaDescription() == nil || *updated.MetaDescription() != testDesc {
		t.Errorf("expected meta description %q, got %+v", testDesc, updated.MetaDescription())
	}
}
