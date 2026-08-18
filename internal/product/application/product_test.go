package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

func baseCreateInput() domain.CreateProductInput {
	return domain.CreateProductInput{
		CategoryID: "cat-1",
		BrandID:    "brand-1",
		Name:       "Máy lạnh Daikin 1.5HP Inverter",
		Slug:       "may-lanh-daikin-15hp-inverter",
		Variants:   []domain.ProductVariantInput{{MPN: "SKU-1", OriginalPrice: 12_000_000}},
	}
}

func TestCreateProduct(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	p, err := CreateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), baseCreateInput())
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
	_, err := CreateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), input)
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
	_, err := CreateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), input)
	if err == nil {
		t.Fatal("expected validation error for zero variants, got nil")
	}
}

func TestUpdateProduct_NotFound(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	_, err := UpdateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), domain.UpdateProductInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateProduct_PartialUpdate(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	created, _ := CreateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), baseCreateInput())

	newName := "Máy lạnh Daikin Updated"
	updated, err := UpdateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), domain.UpdateProductInput{ID: created.ID(), Name: &newName})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name() != newName {
		t.Errorf("expected name updated, got %s", updated.Name())
	}
}

// TestUpdateProduct_ResendingPreExistingOverLongMetaTitle_NoError guards
// against a real regression: legacy products written before the seo
// validation existed can have a metaTitle over MaxMetaTitleLength, and
// every admin save resends the whole form (including untouched fields).
// Editing an unrelated field (OrderIndex here) must not be blocked by that
// pre-existing, unrelated meta title.
func TestUpdateProduct_ResendingPreExistingOverLongMetaTitle_NoError(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	overLongTitle := "This meta title is deliberately far longer than sixty characters allows"
	now := time.Now()
	legacy := domain.RehydrateProduct(
		"legacy-1", "cat-1", "brand-1", "Legacy Product", "legacy-product",
		nil, nil,
		false, domain.ProductStatusPublished, nil, 0,
		&overLongTitle, nil,
		nil, nil,
		nil, nil, nil, nil, nil, "",
		now, now, nil,
	)
	repo.items[legacy.ID()] = legacy

	newOrderIndex := 5
	updated, err := UpdateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), domain.UpdateProductInput{
		ID:         legacy.ID(),
		OrderIndex: &newOrderIndex,
		MetaTitle:  &overLongTitle,
	})
	if err != nil {
		t.Fatalf("expected no error resending an unchanged over-long metaTitle, got %v", err)
	}
	if updated.OrderIndex() != newOrderIndex {
		t.Errorf("expected order index updated, got %d", updated.OrderIndex())
	}

	// Sanity check the guard actually still fires for a genuinely new,
	// over-long value.
	newOverLongTitle := overLongTitle + " and now even longer still"
	_, err = UpdateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), domain.UpdateProductInput{
		ID:        legacy.ID(),
		MetaTitle: &newOverLongTitle,
	})
	if err == nil {
		t.Fatal("expected validation error when actually changing to a new over-long metaTitle")
	}
}

// TestUpdateProduct_EmptyVariantsRejected proves the "at least one variant
// required" invariant also holds on update: explicitly replacing the
// variant tree with an empty list must fail, not silently wipe it.
func TestUpdateProduct_EmptyVariantsRejected(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	created, _ := CreateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), baseCreateInput())

	empty := []domain.ProductVariantInput{}
	_, err := UpdateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), domain.UpdateProductInput{ID: created.ID(), Variants: &empty})
	if err == nil {
		t.Fatal("expected validation error for empty variants, got nil")
	}
}

func TestDeleteAndRestoreProduct(t *testing.T) {
	repo := newFakeProductRepository()
	ctx := context.Background()

	created, _ := CreateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), baseCreateInput())

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

	_, _ = CreateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), baseCreateInput())

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

	created, _ := CreateProduct(ctx, repo, newFakeAttributeDefinitionRepository(), baseCreateInput())

	found, err := GetProductsByIDs(ctx, repo, []string{created.ID(), "missing"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(found) != 1 {
		t.Errorf("expected 1 product found, got %d", len(found))
	}
}
