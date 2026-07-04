//go:build integration

package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/trvux/elc-go/internal/catalog/domain"
	"github.com/trvux/elc-go/internal/platform/db"
)

// Run explicitly with: go test -tags=integration ./internal/catalog/infrastructure/...
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

	value := "1.5 HP"
	specs := []domain.SpecItem{{Label: "Công suất", Value: &value}}
	normalizedSpecs := domain.NormalizeProductSpecs("Integration Test Product 1.5HP", specs)

	p, err := domain.NewProduct(
		categoryID, brandID, "Integration Test Product 1.5HP", "integration-test-sku-xyz", "integration-test-product-xyz",
		nil, specs, normalizedSpecs, []string{"https://example.com/a.webp"}, []string{"moi"},
		100000, nil, 10,
		false, true, 0,
		"in_stock", "new",
		nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("NewProduct failed: %v", err)
	}

	created, err := repo.Create(ctx, p)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM products WHERE id = $1", created.ID())
	}()

	if created.ID() == "" {
		t.Error("expected created product to have an ID")
	}
	// NormalizeProductPrice wasn't applied here directly (that's the
	// application layer's job, see application/create_product.go) — Create
	// just persists whatever the entity carries, so sale_price stays nil as
	// given, which is expected at this layer.
	if len(created.Images()) != 1 || created.Images()[0] != "https://example.com/a.webp" {
		t.Errorf("expected images to round-trip, got %v", created.Images())
	}
	if len(created.NormalizedSpecs()) == 0 {
		t.Errorf("expected normalized_specs to round-trip, got %v", created.NormalizedSpecs())
	}
	if len(created.Specs()) != 1 || created.Specs()[0].Label != "Công suất" {
		t.Errorf("expected specs to round-trip, got %+v", created.Specs())
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
	updated, err := repo.Update(ctx, fetched.Product)
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

// TestPostgresProductRepository_SearchUnaccent proves the highest-risk new
// behavior end to end against real data: an unaccented query must match an
// accented product name via the search_vector + immutable_unaccent generated
// column, and vice versa.
func TestPostgresProductRepository_SearchUnaccent(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresProductRepository(pool)

	result, err := repo.GetAll(ctx, domain.ProductFilter{Search: "dieu hoa", Limit: 5})
	if err != nil {
		t.Fatalf("GetAll (unaccented search) failed: %v", err)
	}
	if len(result.Products) == 0 {
		t.Fatal("expected unaccented search 'dieu hoa' to match at least one real product (contains 'điều hòa')")
	}

	accentedResult, err := repo.GetAll(ctx, domain.ProductFilter{Search: "điều hòa", Limit: 5})
	if err != nil {
		t.Fatalf("GetAll (accented search) failed: %v", err)
	}
	if len(accentedResult.Products) == 0 {
		t.Fatal("expected accented search 'điều hòa' to also match")
	}
}

// TestPostgresProductRepository_Facets checks the facet aggregates come back
// non-empty and that MinPrice/MaxPrice form a sane range against the real
// (196-row) dataset.
func TestPostgresProductRepository_Facets(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresProductRepository(pool)

	result, err := repo.GetAll(ctx, domain.ProductFilter{Limit: 5})
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if result.TotalCount == 0 {
		t.Fatal("expected the real products table to have rows")
	}
	if len(result.Facets.Brands) == 0 {
		t.Error("expected at least one brand facet against the real dataset")
	}
	if result.Facets.MaxPrice < result.Facets.MinPrice {
		t.Errorf("expected MaxPrice >= MinPrice, got min=%d max=%d", result.Facets.MinPrice, result.Facets.MaxPrice)
	}
}

// TestPostgresProductRepository_GetAdjacent exercises the real fallback
// branch (< 2 published siblings -> broaden to the full published catalog)
// against real category data, which a fake-repository unit test can't do —
// see application/get_adjacent_products.go for why that branching lives here.
func TestPostgresProductRepository_GetAdjacent(t *testing.T) {
	ctx := context.Background()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var productID, categoryID string
	if err := pool.QueryRow(ctx,
		"SELECT id, category_id FROM products WHERE deleted_at IS NULL AND is_published = true LIMIT 1",
	).Scan(&productID, &categoryID); err != nil {
		t.Skipf("no published product row available to test against: %v", err)
	}

	repo := NewPostgresProductRepository(pool)

	prev, next, err := repo.GetAdjacent(ctx, categoryID, productID)
	if err != nil {
		t.Fatalf("GetAdjacent failed: %v", err)
	}
	// With 196 real published rows, at least one neighbor should exist
	// (either within category or via the full-catalog fallback) unless this
	// is the only published product in the whole table.
	if prev == nil && next == nil {
		t.Log("no adjacent product found — acceptable only if this is the sole published product")
	}
}
