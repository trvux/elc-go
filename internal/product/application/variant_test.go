package application

import (
	"testing"

	"github.com/trvux/elc-go/internal/product/domain"
)

func TestResolveDefaultVariant_SingleVariantAlwaysDefaulted(t *testing.T) {
	variants := []domain.ProductVariantInput{{MPN: "MPN-1"}}
	resolved, err := resolveDefaultVariant(variants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolved[0].IsDefault {
		t.Error("expected the only variant to be auto-defaulted")
	}
}

func TestResolveDefaultVariant_MultipleNoneMarkedDefaultsFirst(t *testing.T) {
	variants := []domain.ProductVariantInput{{MPN: "MPN-1"}, {MPN: "MPN-2"}}
	resolved, err := resolveDefaultVariant(variants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolved[0].IsDefault || resolved[1].IsDefault {
		t.Errorf("expected only the first variant defaulted, got %+v", resolved)
	}
}

func TestResolveDefaultVariant_RespectsExplicitDefault(t *testing.T) {
	variants := []domain.ProductVariantInput{{MPN: "MPN-1"}, {MPN: "MPN-2", IsDefault: true}}
	resolved, err := resolveDefaultVariant(variants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved[0].IsDefault || !resolved[1].IsDefault {
		t.Errorf("expected the explicitly marked variant to stay default, got %+v", resolved)
	}
}

func TestResolveDefaultVariant_MultipleDefaultsIsValidationError(t *testing.T) {
	variants := []domain.ProductVariantInput{{MPN: "MPN-1", IsDefault: true}, {MPN: "MPN-2", IsDefault: true}}
	_, err := resolveDefaultVariant(variants)
	if err == nil {
		t.Fatal("expected a validation error for two default variants")
	}
}

// TestResolveDefaultVariant_EmptyIsRejected proves the "at least one
// variant required" invariant — Product carries no sku/mpn/price of its
// own, so a create/update with zero variants must be rejected, not
// silently accepted as a no-op.
func TestResolveDefaultVariant_EmptyIsRejected(t *testing.T) {
	_, err := resolveDefaultVariant(nil)
	if err == nil {
		t.Fatal("expected a validation error for zero variants")
	}
}
