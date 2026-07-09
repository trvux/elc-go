package domain

import (
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestNewProductVariant_ValidInput(t *testing.T) {
	v, err := NewProductVariant("prod-1", "FTKB25ZVMV", "SKU-1", nil, true, true, "", nil, nil, 9000000, nil, 0, nil, true, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.StockStatus() != StockStatusInStock {
		t.Errorf("expected default stock status %q, got %q", StockStatusInStock, v.StockStatus())
	}
}

func TestNewProductVariant_RequiresMPN(t *testing.T) {
	_, err := NewProductVariant("prod-1", "", "SKU-1", nil, true, true, "", nil, nil, 0, nil, 0, nil, true, 0, nil)
	assertVariantValidationError(t, err)
}

func TestNewProductVariant_RequiresSKU(t *testing.T) {
	_, err := NewProductVariant("prod-1", "MPN-1", "", nil, true, true, "", nil, nil, 0, nil, 0, nil, true, 0, nil)
	assertVariantValidationError(t, err)
}

func TestNewProductVariant_RejectsInvalidStockStatus(t *testing.T) {
	_, err := NewProductVariant("prod-1", "MPN-1", "SKU-1", nil, true, true, "backordered", nil, nil, 0, nil, 0, nil, true, 0, nil)
	assertVariantValidationError(t, err)
}

func TestProductVariant_DisplayPrice(t *testing.T) {
	sale := int64(8000000)
	v, err := NewProductVariant("prod-1", "MPN-1", "SKU-1", nil, true, true, "", nil, nil, 9000000, &sale, 0, nil, true, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.DisplayPrice() != sale {
		t.Errorf("expected DisplayPrice to prefer sale price, got %d", v.DisplayPrice())
	}

	v2, err := NewProductVariant("prod-1", "MPN-2", "SKU-2", nil, true, true, "", nil, nil, 9000000, nil, 0, nil, true, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v2.DisplayPrice() != 9000000 {
		t.Errorf("expected DisplayPrice to fall back to original price, got %d", v2.DisplayPrice())
	}
}

func TestNewProductLine_RequiresBrandCodeName(t *testing.T) {
	_, err := NewProductLine(nil, nil, "FTKZ", "Dòng Inverter siêu cao cấp", 4, nil, nil)
	assertVariantValidationError(t, err)

	brand := "brand-1"
	_, err = NewProductLine(&brand, nil, "", "Dòng Inverter siêu cao cấp", 4, nil, nil)
	assertVariantValidationError(t, err)

	_, err = NewProductLine(&brand, nil, "FTKZ", "", 4, nil, nil)
	assertVariantValidationError(t, err)
}

func TestNewProductLine_ValidInput(t *testing.T) {
	brand := "brand-1"
	line, err := NewProductLine(&brand, nil, "FTKZ", "Dòng Inverter siêu cao cấp", 4, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line.Code() != "FTKZ" || line.TierRank() != 4 {
		t.Errorf("unexpected product line: %+v", line)
	}
}

func assertVariantValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
}
