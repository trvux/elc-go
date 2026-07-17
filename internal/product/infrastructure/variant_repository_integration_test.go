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
func TestPostgresProductRepository_VariantTree(t *testing.T) {
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
		categoryID, brandID, "Integration Test Split AC", "integration-test-split-ac",
		nil, nil, nil,
		false, true, 0,
		"",
		nil, nil,
		nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("NewProduct failed: %v", err)
	}

	options := []domain.ProductOptionInput{
		{Name: "Màu sắc", Values: []string{"Đỏ", "Đen"}},
	}

	sale := int64(8500000)
	variants := []domain.ProductVariantInput{
		// index 0: indoor unit component, never sold standalone.
		{MPN: "ZZTEST-INDOOR-25", IsComponentOnly: true, IsActive: true, OriginalPrice: 5000000},
		// index 1: outdoor unit component, never sold standalone.
		{MPN: "ZZTEST-OUTDOOR-25", IsComponentOnly: true, IsActive: true, OriginalPrice: 4000000},
		// index 2: the sellable bundle, red — references option index by name/value and both components above.
		{
			MPN: "ZZTEST-SET-RED", IsDefault: true, IsActive: true,
			OriginalPrice: 9500000, SalePrice: &sale,
			OptionSelections: []domain.VariantOptionSelection{{OptionName: "Màu sắc", Value: "Đỏ"}},
			Components: []domain.VariantComponentInput{
				{ComponentIndex: 0, Quantity: 1, Role: strPtr("Dàn lạnh")},
				{ComponentIndex: 1, Quantity: 1, Role: strPtr("Dàn nóng")},
			},
		},
		// index 3: the sellable bundle, black — no sale price, tests price_min/price_max spread.
		{
			MPN: "ZZTEST-SET-BLACK", IsActive: true,
			OriginalPrice:    9700000,
			OptionSelections: []domain.VariantOptionSelection{{OptionName: "Màu sắc", Value: "Đen"}},
			Components: []domain.VariantComponentInput{
				{ComponentIndex: 0, Quantity: 1},
				{ComponentIndex: 1, Quantity: 1},
			},
		},
	}

	created, err := repo.Create(ctx, p, nil, options, variants, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM products WHERE id = $1", created.ID())
	}()

	// Denormalized cache: default variant is the red bundle (index 2, sale
	// price 8.5M); price_min/price_max span both sellable bundles AND both
	// standalone-false components (is_standalone doesn't affect the
	// min/max aggregate, only is_active — components have no sale price so
	// their original_price of 5M/4M pulls price_min down to 4M).
	if created.DefaultVariantID() == nil {
		t.Fatal("expected default_variant_id to be set")
	}
	if created.DisplayPrice() == nil || *created.DisplayPrice() != sale {
		t.Errorf("expected display_price %d, got %v", sale, created.DisplayPrice())
	}
	if created.PriceMin() == nil || *created.PriceMin() != 4000000 {
		t.Errorf("expected price_min 4000000, got %v", created.PriceMin())
	}
	if created.PriceMax() == nil || *created.PriceMax() != 9700000 {
		t.Errorf("expected price_max 9700000, got %v", created.PriceMax())
	}

	fetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if len(fetched.Options) != 1 || len(fetched.Options[0].Values) != 2 {
		t.Fatalf("expected 1 option with 2 values, got %+v", fetched.Options)
	}
	if len(fetched.Variants) != 4 {
		t.Fatalf("expected 4 variants, got %d", len(fetched.Variants))
	}

	var bundleRed, indoorUnit *domain.ProductVariant
	for _, v := range fetched.Variants {
		switch v.MPN() {
		case "ZZTEST-SET-RED":
			bundleRed = v
		case "ZZTEST-INDOOR-25":
			indoorUnit = v
		}
	}
	if bundleRed == nil || indoorUnit == nil {
		t.Fatalf("expected to find both the red bundle and the indoor unit component, got %+v", fetched.Variants)
	}
	if len(bundleRed.OptionValueIDs()) != 1 {
		t.Errorf("expected red bundle to have 1 linked option value, got %v", bundleRed.OptionValueIDs())
	}
	if len(bundleRed.Components()) != 2 {
		t.Fatalf("expected red bundle to have 2 components, got %+v", bundleRed.Components())
	}
	if indoorUnit.IsStandalone() {
		t.Error("expected the indoor unit component to have is_standalone=false")
	}
	if !bundleRed.IsStandalone() {
		t.Error("expected the sellable bundle to have is_standalone=true")
	}

	// Update: replace the whole tree with a single, simpler variant — proves
	// the wholesale delete-then-reinsert + cache recompute round trip.
	singleVariant := []domain.ProductVariantInput{
		{MPN: "ZZTEST-SET-SIMPLE", OriginalPrice: 9000000, IsActive: true},
	}
	emptyOptions := []domain.ProductOptionInput{}
	updated, err := repo.Update(ctx, fetched.Product, nil, &emptyOptions, &singleVariant, nil)
	if err != nil {
		t.Fatalf("Update (replace variant tree) failed: %v", err)
	}
	// No variant in this update was marked IsDefault (the repository itself
	// doesn't auto-default a lone variant — see the boundary note below), so
	// display_price stays nil while price_min/price_max (which don't depend
	// on is_default) still reflect the one active variant.
	if updated.DisplayPrice() != nil {
		t.Errorf("expected display_price nil (no default variant set), got %v", *updated.DisplayPrice())
	}

	refetched, err := repo.GetByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if len(refetched.Variants) != 1 || refetched.Variants[0].MPN() != "ZZTEST-SET-SIMPLE" {
		t.Fatalf("expected exactly 1 replaced variant, got %+v", refetched.Variants)
	}
	if len(refetched.Options) != 0 {
		t.Errorf("expected options to be cleared, got %+v", refetched.Options)
	}
	// The repository itself does NOT auto-default a lone variant — that rule
	// lives in application.resolveDefaultVariant (see application/
	// create_product.go), deliberately not duplicated at this layer. Since
	// this test called the repository directly with IsDefault left false,
	// default_variant_id/display_price stay nil while price_min/price_max
	// still reflect the one active variant regardless of default status.
	if updated.DefaultVariantID() != nil {
		t.Errorf("expected no default_variant_id (repository doesn't auto-default), got %v", *updated.DefaultVariantID())
	}
	if updated.PriceMin() == nil || *updated.PriceMin() != 9000000 || updated.PriceMax() == nil || *updated.PriceMax() != 9000000 {
		t.Errorf("expected price_min/price_max both 9000000, got min=%v max=%v", updated.PriceMin(), updated.PriceMax())
	}
}

func strPtr(s string) *string { return &s }
