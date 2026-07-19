//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/news/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/news/infrastructure/...
func TestPostgresNewsRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresNewsRepository(pool)

	n, err := domain.NewNews(
		"Integration Test News", "integration-test-news-xyz", nil,
		nil, "", nil, nil, false, nil, nil, 999,
	)
	if err != nil {
		t.Fatalf("NewNews failed: %v", err)
	}

	created, err := repo.Create(ctx, n, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM news WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created news to have an ID")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Slug() != "integration-test-news-xyz" {
		t.Fatalf("expected fetched news to match created one, got %+v", fetched)
	}

	bySlug, err := repo.GetBySlug(ctx, "integration-test-news-xyz")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if bySlug == nil || bySlug.ID() != created.ID() {
		t.Errorf("expected GetBySlug to find the created news, got %+v", bySlug)
	}

	if err := fetched.UpdateTitle("Integration Test News Updated"); err != nil {
		t.Fatalf("UpdateTitle failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title() != "Integration Test News Updated" {
		t.Errorf("expected updated title, got %s", updated.Title())
	}

	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted news to not be found by GetByID")
	}
	if found, _ := repo.GetBySlug(ctx, "integration-test-news-xyz"); found != nil {
		t.Error("expected soft-deleted news to not be found by GetBySlug")
	}

	if err := repo.Restore(ctx, created.ID()); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found == nil {
		t.Error("expected restored news to be found again")
	}
}

// TestPostgresNewsRepository_ResurrectOnSlugReuse exercises the mandatory
// (not business-optional) resurrect: news.slug is a plain UNIQUE constraint
// (unlike brand/group/category's partial index), so re-creating with the
// same slug while the old row is soft-deleted MUST reuse the same row/ID —
// a plain INSERT would otherwise violate the constraint outright. See
// docs/news.md.
func TestPostgresNewsRepository_ResurrectOnSlugReuse(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresNewsRepository(pool)

	n1, _ := domain.NewNews("Resurrect Test", "integration-test-resurrect-news-xyz", nil, nil, "", nil, nil, false, nil, nil, 0)
	created1, err := repo.Create(ctx, n1, nil)
	if err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM news WHERE id = $1", created1.ID())
	}()

	if err := repo.SoftDelete(ctx, created1.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	n2, _ := domain.NewNews("Resurrect Test Again", "integration-test-resurrect-news-xyz", nil, nil, "", nil, nil, false, nil, nil, 0)
	created2, err := repo.Create(ctx, n2, nil)
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
