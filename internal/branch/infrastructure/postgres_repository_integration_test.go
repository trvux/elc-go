//go:build integration

package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/trvux/elc-go/internal/branch/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/branch/infrastructure/...
func TestPostgresBranchRepository_CRUD(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresBranchRepository(pool)

	desc := json.RawMessage(`{"text": "Chi nhánh kiểm thử"}`)
	images := []domain.ImageAsset{{URL: "https://example.com/logo.png"}}

	b, err := domain.NewBranch(
		"ELC Integration Q12", "elc-integration-q12-xyz", "123 Le Loi, Q12", "0901122334",
		"q12@elc.vn", "https://maps.google.com/q12", "<iframe></iframe>",
		desc, images, true, 999, nil, nil,
	)
	if err != nil {
		t.Fatalf("NewBranch failed: %v", err)
	}

	created, err := repo.Create(ctx, b)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		// Clean up in case test fails
		_ = repo.Delete(ctx, created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created branch to have an ID")
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Slug() != "elc-integration-q12-xyz" {
		t.Errorf("expected fetched branch to match created one, got %+v", fetched)
	}

	bySlug, err := repo.GetBySlug(ctx, "elc-integration-q12-xyz")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if bySlug == nil || bySlug.ID() != created.ID() {
		t.Errorf("expected GetBySlug to find the created branch, got %+v", bySlug)
	}

	// Verify count works
	count, err := repo.Count(ctx, domain.BranchFilter{Search: "ELC Integration Q12"})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count to be 1, got %d", count)
	}

	// Update branch
	if err := fetched.UpdateName("ELC Integration Q12 Updated"); err != nil {
		t.Fatalf("UpdateName failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name() != "ELC Integration Q12 Updated" {
		t.Errorf("expected updated name, got %s", updated.Name())
	}

	// Hard delete the branch
	if err := repo.Delete(ctx, created.ID()); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's no longer found
	deletedFetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID post-delete failed: %v", err)
	}
	if deletedFetched != nil {
		t.Error("expected branch to be deleted and not found")
	}
}
