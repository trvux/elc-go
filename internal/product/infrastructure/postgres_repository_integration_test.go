//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/product/domain"
)

// Run explicitly with: go test -tags=integration ./internal/product/infrastructure/...
func TestPostgresProductRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var categoryID, brandID string
	if err := pool.QueryRow(ctx, "SELECT id FROM categories LIMIT 1").Scan(&categoryID); err != nil {
		t.Skipf("no category row available to test against: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT id FROM brands LIMIT 1").Scan(&brandID); err != nil {
		t.Skipf("no brand row available to test against: %v", err)
	}

	repo := NewPostgresProductRepository(pool)

	p, err := domain.NewProduct(
		categoryID, brandID, "Integration Test Product 1.5HP", "integration-test-product-xyz",
		nil, []domain.ImageAsset{{URL: "https://example.com/a.webp"}}, []string{"moi"},
		false, true, 0,
		"new",
		nil, nil,
		nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("NewProduct failed: %v", err)
	}

	variants := []domain.ProductVariantInput{
		{MPN: "integration-test-mpn-xyz", OriginalPrice: 100000, IsActive: true, IsDefault: true},
	}
	created, err := repo.Create(ctx, p, nil, nil, variants, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM products WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created product to have an ID")
	}
	if len(created.Images()) != 1 || created.Images()[0].URL != "https://example.com/a.webp" {
		t.Errorf("expected images to round-trip, got %v", created.Images())
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched == nil || fetched.Slug() != "integration-test-product-xyz" {
		t.Fatalf("expected fetched product to match created one, got %+v", fetched)
	}
	if fetched.Category == nil || fetched.Category.ID != categoryID {
		t.Errorf("expected Category to be joined with id %s, got %+v", categoryID, fetched.Category)
	}
	if fetched.Brand == nil || fetched.Brand.ID != brandID {
		t.Errorf("expected Brand to be joined with id %s, got %+v", brandID, fetched.Brand)
	}

	bySlug, err := repo.GetBySlug(ctx, "integration-test-product-xyz")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if bySlug == nil || bySlug.ID() != created.ID() {
		t.Errorf("expected GetBySlug to find the created product, got %+v", bySlug)
	}

	byIDs, err := repo.GetByIDs(ctx, []string{created.ID(), "00000000-0000-0000-0000-000000000000"})
	if err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}
	if len(byIDs) != 1 || byIDs[0].ID() != created.ID() {
		t.Errorf("expected GetByIDs to find exactly the created product, got %+v", byIDs)
	}

	if err := fetched.Product.UpdateName("Integration Test Product 1.5HP Updated"); err != nil {
		t.Fatalf("UpdateName failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched.Product, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name() != "Integration Test Product 1.5HP Updated" {
		t.Errorf("expected updated name, got %s", updated.Name())
	}

	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted product to not be found by GetByID")
	}
	if found, _ := repo.GetBySlug(ctx, "integration-test-product-xyz"); found != nil {
		t.Error("expected soft-deleted product to not be found by GetBySlug")
	}

	if err := repo.Restore(ctx, created.ID()); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found == nil {
		t.Error("expected restored product to be found again")
	}
}
