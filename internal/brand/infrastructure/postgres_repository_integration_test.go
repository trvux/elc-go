//go:build integration

package infrastructure

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/brand/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/brand/infrastructure/...
func TestPostgresBrandRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresBrandRepository(pool)

	content := json.RawMessage(`{"type":"doc","content":[]}`)
	faq := []domain.FAQItem{{Question: "Bao hanh may nam?", Answer: "2 nam"}}

	b, err := domain.NewBrand(
		"Integration Test Brand", "integration-test-brand-xyz", "https://example.com/logo.png",
		nil, nil, false, 999, content, faq,
	)
	if err != nil {
		t.Fatalf("NewBrand failed: %v", err)
	}

	created, err := repo.Create(ctx, b)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM brands WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created brand to have an ID")
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
	if fetched == nil || fetched.Slug() != "integration-test-brand-xyz" {
		t.Errorf("expected fetched brand to match created one, got %+v", fetched)
	}

	bySlug, err := repo.GetBySlug(ctx, "integration-test-brand-xyz")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if bySlug == nil || bySlug.ID() != created.ID() {
		t.Errorf("expected GetBySlug to find the created brand, got %+v", bySlug)
	}

	if err := fetched.UpdateName("Integration Test Brand Updated"); err != nil {
		t.Fatalf("UpdateName failed: %v", err)
	}
	updated, err := repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name() != "Integration Test Brand Updated" {
		t.Errorf("expected updated name, got %s", updated.Name())
	}

	// Soft delete then verify it's excluded from GetByID and GetBySlug.
	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found != nil {
		t.Error("expected soft-deleted brand to not be found by GetByID")
	}
	if found, _ := repo.GetBySlug(ctx, "integration-test-brand-xyz"); found != nil {
		t.Error("expected soft-deleted brand to not be found by GetBySlug")
	}

	if err := repo.Restore(ctx, created.ID()); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if found, _ := repo.GetByID(ctx, created.ID()); found == nil {
		t.Error("expected restored brand to be found again")
	}

	// Soft delete again to free up the slug for the next assertion.
	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("second SoftDelete failed: %v", err)
	}

	// Unlike service_groups, brands.slug is only unique among non-deleted
	// rows — re-creating with the same slug while the old row stays
	// soft-deleted must succeed as a fresh insert, not a conflict/resurrect.
	reuseSlugInput, err := domain.NewBrand(
		"Another Brand Reusing Slug", "integration-test-brand-xyz", "",
		nil, nil, false, 1, nil, nil,
	)
	if err != nil {
		t.Fatalf("NewBrand (slug reuse) failed: %v", err)
	}
	reused, err := repo.Create(ctx, reuseSlugInput)
	if err != nil {
		t.Fatalf("Create (slug reuse) failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM brands WHERE id = $1", reused.ID())
	}()
	if reused.ID() == created.ID() {
		t.Error("expected slug reuse to insert a new row, not resurrect the soft-deleted one")
	}
}

// TestPostgresBrandRepository_SoftDeleteLeavesProductBrandIDIntact documents
// a deliberate deviation from the service-group precedent: products.brand_id
// is NOT NULL (confirmed via psql), so soft-deleting a brand cannot null out
// referencing products the way service-group's SoftDelete nulls
// services.group_id (nullable there). See docs/brand.md.
func TestPostgresBrandRepository_SoftDeleteLeavesProductBrandIDIntact(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresBrandRepository(pool)

	b, err := domain.NewBrand("FK Cleanup Test Brand", "fk-cleanup-test-brand-xyz", "", nil, nil, false, 0, nil, nil)
	if err != nil {
		t.Fatalf("NewBrand failed: %v", err)
	}
	created, err := repo.Create(ctx, b)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM brands WHERE id = $1", created.ID())
	}()

	var categoryID string
	if err := pool.QueryRow(ctx, "SELECT id FROM categories LIMIT 1").Scan(&categoryID); err != nil {
		t.Skipf("no category row available to attach a test product to: %v", err)
	}

	var productID string
	err = pool.QueryRow(ctx, `
		INSERT INTO products (category_id, name, sku, slug, brand_id)
		VALUES ($1, 'FK Cleanup Test Product', 'fk-cleanup-test-sku-xyz', 'fk-cleanup-test-product-xyz', $2)
		RETURNING id`,
		categoryID, created.ID(),
	).Scan(&productID)
	if err != nil {
		t.Fatalf("failed to insert test product: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM products WHERE id = $1", productID)
	}()

	if err := repo.SoftDelete(ctx, created.ID()); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	var brandID string
	if err := pool.QueryRow(ctx, "SELECT brand_id FROM products WHERE id = $1", productID).Scan(&brandID); err != nil {
		t.Fatalf("failed to read back product: %v", err)
	}
	if brandID != created.ID() {
		t.Errorf("expected product.brand_id to remain %s after brand soft delete, got %s", created.ID(), brandID)
	}
}
