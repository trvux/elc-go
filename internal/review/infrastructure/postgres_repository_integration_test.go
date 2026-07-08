//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/review/domain"
)

// Run explicitly with: go test -tags=integration ./internal/review/infrastructure/...
// Not part of the normal `go test ./...` run — this hits the real Postgres,
// so it must never run implicitly (see ARCHITECTURE.md section 10/17).
func TestPostgresReviewRepository_CRUD(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var productID string
	if err := pool.QueryRow(ctx, "SELECT id FROM products WHERE deleted_at IS NULL LIMIT 1").Scan(&productID); err != nil {
		t.Skipf("no product available to attach a test review to: %v", err)
	}

	repo := NewPostgresReviewRepository(pool)

	review, err := domain.NewReview(&productID, nil, 5, "Integration test review — máy chạy tốt.", "Integration Tester", nil, nil, nil)
	if err != nil {
		t.Fatalf("NewReview failed: %v", err)
	}

	created, err := repo.Create(ctx, review)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_ = repo.Delete(ctx, created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created review to have an ID")
	}
	if !created.IsPublished() {
		t.Error("expected clean review to be published by default")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.ReviewerName() != "Integration Tester" {
		t.Errorf("expected fetched review to match created one, got %+v", fetched)
	}

	fetched.SetPublished(false)
	updated, err := repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.IsPublished() {
		t.Error("expected review to be hidden after Update")
	}

	// Republish and confirm the summary picks it up.
	updated.SetPublished(true)
	if _, err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update (republish) failed: %v", err)
	}

	summary, err := repo.GetSummary(ctx, &productID, nil)
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}
	if summary.Count < 1 {
		t.Errorf("expected summary count >= 1, got %d", summary.Count)
	}

	count, err := repo.Count(ctx, domain.ReviewFilter{ProductID: &productID})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count < 1 {
		t.Errorf("expected count >= 1 for productID filter, got %d", count)
	}

	if err := repo.Delete(ctx, created.ID()); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	deletedFetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID post-delete failed: %v", err)
	}
	if deletedFetched != nil {
		t.Error("expected review to be deleted and not found")
	}
}
