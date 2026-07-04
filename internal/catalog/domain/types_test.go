package domain

import (
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func newTestProduct(t *testing.T) *Product {
	t.Helper()
	p, err := NewProduct(
		"cat-1", "brand-1", "Máy lạnh Daikin", "SKU-1", "may-lanh-daikin",
		nil, nil, nil, nil, nil,
		1000, nil, 0,
		false, true, 0,
		"", "",
		nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return p
}

func TestNewProduct(t *testing.T) {
	t.Run("valid input creates a product with defaults", func(t *testing.T) {
		p := newTestProduct(t)
		if p.Name() != "Máy lạnh Daikin" || p.Slug() != "may-lanh-daikin" {
			t.Errorf("unexpected product: %+v", p)
		}
		if p.StockStatus() != "in_stock" {
			t.Errorf("expected default stock status 'in_stock', got %s", p.StockStatus())
		}
		if p.Condition() != "new" {
			t.Errorf("expected default condition 'new', got %s", p.Condition())
		}
		if p.IsDeleted() {
			t.Error("expected new product to not be deleted")
		}
	})

	t.Run("empty name fails validation", func(t *testing.T) {
		_, err := NewProduct("cat-1", "brand-1", "", "sku", "slug", nil, nil, nil, nil, nil, 0, nil, 0, false, true, 0, "", "", nil, nil, nil, nil)
		assertValidationError(t, err)
	})

	t.Run("empty slug fails validation", func(t *testing.T) {
		_, err := NewProduct("cat-1", "brand-1", "name", "sku", "", nil, nil, nil, nil, nil, 0, nil, 0, false, true, 0, "", "", nil, nil, nil, nil)
		assertValidationError(t, err)
	})

	t.Run("empty sku fails validation", func(t *testing.T) {
		_, err := NewProduct("cat-1", "brand-1", "name", "", "slug", nil, nil, nil, nil, nil, 0, nil, 0, false, true, 0, "", "", nil, nil, nil, nil)
		assertValidationError(t, err)
	})

	t.Run("empty category_id fails validation", func(t *testing.T) {
		_, err := NewProduct("", "brand-1", "name", "sku", "slug", nil, nil, nil, nil, nil, 0, nil, 0, false, true, 0, "", "", nil, nil, nil, nil)
		assertValidationError(t, err)
	})

	t.Run("empty brand_id fails validation", func(t *testing.T) {
		_, err := NewProduct("cat-1", "", "name", "sku", "slug", nil, nil, nil, nil, nil, 0, nil, 0, false, true, 0, "", "", nil, nil, nil, nil)
		assertValidationError(t, err)
	})

	t.Run("invalid condition fails validation", func(t *testing.T) {
		_, err := NewProduct("cat-1", "brand-1", "name", "sku", "slug", nil, nil, nil, nil, nil, 0, nil, 0, false, true, 0, "", "refurbished", nil, nil, nil, nil)
		assertValidationError(t, err)
	})
}

func assertValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
}

func TestProduct_UpdateName(t *testing.T) {
	p := newTestProduct(t)

	if err := p.UpdateName("Máy lạnh Daikin Updated"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "Máy lạnh Daikin Updated" {
		t.Errorf("expected updated name, got %s", p.Name())
	}
	if err := p.UpdateName(""); err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestProduct_UpdateSpecsKeepsInSync(t *testing.T) {
	p := newTestProduct(t)

	value := "1.5 HP"
	specs := []SpecItem{{Label: "Công suất", Value: &value}}
	normalized := NormalizeProductSpecs(p.Name(), specs)

	p.UpdateSpecs(specs, normalized)

	if len(p.Specs()) != 1 {
		t.Fatalf("expected specs to be set, got %+v", p.Specs())
	}
	if len(p.NormalizedSpecs()) == 0 {
		t.Fatalf("expected normalizedSpecs to be populated, got %+v", p.NormalizedSpecs())
	}
}

func TestProduct_UpdatePricing(t *testing.T) {
	p := newTestProduct(t)

	original, sale, discount := NormalizeProductPrice(100000, 0, 10)
	p.UpdatePricing(original, &sale, discount)

	if p.OriginalPrice() != 100000 {
		t.Errorf("expected original price 100000, got %d", p.OriginalPrice())
	}
	if p.SalePrice() == nil || *p.SalePrice() != 90000 {
		t.Errorf("expected sale price 90000, got %v", p.SalePrice())
	}
	if p.DiscountPercent() != 10 {
		t.Errorf("expected discount percent 10, got %v", p.DiscountPercent())
	}
}

func TestProduct_MarkDeletedAndRestore(t *testing.T) {
	p := newTestProduct(t)

	p.MarkDeleted(p.CreatedAt())
	if !p.IsDeleted() {
		t.Error("expected product to be deleted")
	}

	p.Restore()
	if p.IsDeleted() {
		t.Error("expected product to be restored")
	}
	if p.DeletedAt() != nil {
		t.Error("expected deletedAt to be cleared after restore")
	}
}

func TestProduct_SetFeaturedPublishedReorder(t *testing.T) {
	p := newTestProduct(t)

	p.SetFeatured(true)
	if !p.IsFeatured() {
		t.Error("expected product to be featured")
	}

	p.SetPublished(false)
	if p.IsPublished() {
		t.Error("expected product to be unpublished")
	}

	p.Reorder(5)
	if p.OrderIndex() != 5 {
		t.Errorf("expected orderIndex 5, got %d", p.OrderIndex())
	}
}

func TestProduct_UpdateCondition(t *testing.T) {
	p := newTestProduct(t)

	if err := p.UpdateCondition("used"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Condition() != "used" {
		t.Errorf("expected condition 'used', got %s", p.Condition())
	}

	if err := p.UpdateCondition("refurbished"); err == nil {
		t.Fatal("expected validation error for invalid condition")
	}
}
