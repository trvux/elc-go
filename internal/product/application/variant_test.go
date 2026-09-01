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

// TestResolveDefaultVariant_SkipsComponentOnlyWhenAutoDefaulting proves the
// auto-default path (none marked default) skips a leading component-only
// variant in favor of the first standalone one, instead of blindly
// rejecting index 0 — see docs/rfc/2026-09-02-backend-code-review-round2.md
// §3.3 and the code-review finding on the first version of this fix.
func TestResolveDefaultVariant_SkipsComponentOnlyWhenAutoDefaulting(t *testing.T) {
	variants := []domain.ProductVariantInput{
		{MPN: "COMPONENT", IsComponentOnly: true},
		{MPN: "MPN-1"},
		{MPN: "MPN-2"},
	}
	resolved, err := resolveDefaultVariant(variants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved[0].IsDefault {
		t.Error("component-only variant must never be auto-defaulted")
	}
	if !resolved[1].IsDefault || resolved[2].IsDefault {
		t.Errorf("expected the first standalone variant (index 1) defaulted, got %+v", resolved)
	}
}

// TestResolveDefaultVariant_AllComponentOnlyIsRejected proves that when
// every variant is component-only, there is genuinely no valid default to
// pick and the request is rejected (not silently defaulted to a
// component-only variant).
func TestResolveDefaultVariant_AllComponentOnlyIsRejected(t *testing.T) {
	variants := []domain.ProductVariantInput{
		{MPN: "COMPONENT-1", IsComponentOnly: true},
		{MPN: "COMPONENT-2", IsComponentOnly: true},
	}
	_, err := resolveDefaultVariant(variants)
	if err == nil {
		t.Fatal("expected a validation error when every variant is component-only")
	}
}

// TestResolveDefaultVariant_ExplicitComponentOnlyDefaultIsRejected proves an
// explicitly-marked component-only default is still rejected even when a
// standalone alternative exists — auto-picking a different variant instead
// would silently ignore the caller's explicit (if wrong) request.
func TestResolveDefaultVariant_ExplicitComponentOnlyDefaultIsRejected(t *testing.T) {
	variants := []domain.ProductVariantInput{
		{MPN: "COMPONENT", IsComponentOnly: true, IsDefault: true},
		{MPN: "MPN-1"},
	}
	_, err := resolveDefaultVariant(variants)
	if err == nil {
		t.Fatal("expected a validation error for an explicit component-only default")
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
