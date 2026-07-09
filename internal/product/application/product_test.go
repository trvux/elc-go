package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

func baseCreateInput() domain.CreateProductInput {
	return domain.CreateProductInput{
		CategoryID: "cat-1",
		BrandID:    "brand-1",
		Name:       "Máy lạnh Daikin 1.5HP Inverter",
		Slug:       "may-lanh-daikin-15hp-inverter",
		Variants:   []domain.ProductVariantInput{{MPN: "SKU-1"}},
	}
}

func TestCreateProduct(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	p, err := CreateProduct(ctx, repo, baseCreateInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p.ID() == "" {
		t.Error("expected an ID after create")
	}
}

func TestCreateProduct_ValidationError(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	input := baseCreateInput()
	input.Name = ""
	_, err := CreateProduct(ctx, repo, input)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

// TestCreateProduct_NoVariants proves the "at least one variant required"
// invariant — Product itself carries no sku/mpn/price, so a product with
// zero variants must be rejected rather than silently accepted.
func TestCreateProduct_NoVariants(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	input := baseCreateInput()
	input.Variants = nil
	_, err := CreateProduct(ctx, repo, input)
	if err == nil {
		t.Fatal("expected validation error for zero variants, got nil")
	}
}

// TestCreateProduct_NormalizesSpecsAtWriteTime is the key behavioral test for
// the write-time spec normalization design: normalized_specs must be
// populated from the name (capacity facet) even though no explicit specs
// were provided.
func TestCreateProduct_NormalizesSpecsAtWriteTime(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	p, err := CreateProduct(ctx, repo, baseCreateInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found := false
	for _, f := range p.NormalizedSpecs() {
		if f == "Công suất::1.5 HP" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected normalized_specs to contain 'Công suất::1.5 HP', got %+v", p.NormalizedSpecs())
	}
}

func TestUpdateProduct_NotFound(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	_, err := UpdateProduct(ctx, repo, domain.UpdateProductInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateProduct_PartialUpdate(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	created, _ := CreateProduct(ctx, repo, baseCreateInput())

	newName := "Máy lạnh Daikin Updated"
	updated, err := UpdateProduct(ctx, repo, domain.UpdateProductInput{ID: created.ID(), Name: &newName})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name() != newName {
		t.Errorf("expected name updated, got %s", updated.Name())
	}
}

// TestUpdateProduct_EmptyVariantsRejected proves the "at least one variant
// required" invariant also holds on update: explicitly replacing the
// variant tree with an empty list must fail, not silently wipe it.
func TestUpdateProduct_EmptyVariantsRejected(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	created, _ := CreateProduct(ctx, repo, baseCreateInput())

	empty := []domain.ProductVariantInput{}
	_, err := UpdateProduct(ctx, repo, domain.UpdateProductInput{ID: created.ID(), Variants: &empty})
	if err == nil {
		t.Fatal("expected validation error for empty variants, got nil")
	}
}

// TestUpdateProduct_RenamingRecomputesNormalizedSpecs proves the "recompute
// on name-only change" rule in update_product.go — normalized_specs must
// pick up a capacity facet from a new name even when Specs isn't part of the
// same request.
func TestUpdateProduct_RenamingRecomputesNormalizedSpecs(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	created, _ := CreateProduct(ctx, repo, baseCreateInput())

	newName := "Máy lạnh Daikin 2HP Inverter"
	updated, err := UpdateProduct(ctx, repo, domain.UpdateProductInput{ID: created.ID(), Name: &newName})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found := false
	for _, f := range updated.NormalizedSpecs() {
		if f == "Công suất::2 HP" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected normalized_specs to reflect the new name's capacity, got %+v", updated.NormalizedSpecs())
	}
}

func TestDeleteAndRestoreProduct(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	created, _ := CreateProduct(ctx, repo, baseCreateInput())

	if err := DeleteProduct(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetProductByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted product to not be found by GetByID")
	}

	if err := RestoreProduct(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetProductByID(ctx, repo, created.ID()); found == nil {
		t.Error("expected restored product to be found again")
	}
}

func TestListProducts(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	_, _ = CreateProduct(ctx, repo, baseCreateInput())

	result, err := ListProducts(ctx, repo, domain.ProductFilter{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("expected 1 product, got %d", result.TotalCount)
	}
}

func TestGetProductsByIDs(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	created, _ := CreateProduct(ctx, repo, baseCreateInput())

	found, err := GetProductsByIDs(ctx, repo, []string{created.ID(), "missing"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(found) != 1 {
		t.Errorf("expected 1 product found, got %d", len(found))
	}
}
