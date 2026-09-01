package domain

import (
	"testing"
	"time"
)

func TestService_SalePrice(t *testing.T) {
	tests := []struct {
		name            string
		originalPrice   *int64
		discountPercent *int
		want            *int64
	}{
		{name: "no price set", originalPrice: nil, discountPercent: nil, want: nil},
		{name: "price without discount", originalPrice: ptrInt64(100000), discountPercent: nil, want: nil},
		{name: "20 percent off 100000", originalPrice: ptrInt64(100000), discountPercent: ptrInt(20), want: ptrInt64(80000)},
		{name: "0 percent off", originalPrice: ptrInt64(50000), discountPercent: ptrInt(0), want: ptrInt64(50000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := RehydrateService(
				"id-1", "Title", "slug", nil, nil,
				tt.originalPrice, tt.discountPercent, nil, nil, nil, nil,
				nil, nil, nil, false, true, 0,
				time.Now(), time.Now(), nil,
			)

			got := s.SalePrice()
			if (got == nil) != (tt.want == nil) {
				t.Fatalf("SalePrice() = %v, want %v", got, tt.want)
			}
			if got != nil && *got != *tt.want {
				t.Errorf("SalePrice() = %d, want %d", *got, *tt.want)
			}
		})
	}
}

func TestService_UpdatePricing_NeverGoesStale(t *testing.T) {
	s := RehydrateService(
		"id-1", "Title", "slug", nil, nil,
		ptrInt64(100000), ptrInt(10), nil, nil, nil, nil,
		nil, nil, nil, false, true, 0,
		time.Now(), time.Now(), nil,
	)

	if *s.SalePrice() != 90000 {
		t.Fatalf("expected initial sale price 90000, got %d", *s.SalePrice())
	}

	// Simulate an update that only touches discountPercent — the exact case
	// that went stale in the old TS code. Update() resolves OriginalPrice/
	// DiscountPercent together internally (see its doc comment), so a
	// request that only sets DiscountPercent still can't go stale.
	if err := s.Update(UpdateServiceInput{DiscountPercent: ptrInt(50)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *s.SalePrice() != 50000 {
		t.Errorf("expected recomputed sale price 50000, got %d", *s.SalePrice())
	}
}

func TestService_Update_RejectsInvalidDiscountPercent(t *testing.T) {
	s := RehydrateService(
		"id-1", "Title", "slug", nil, nil,
		ptrInt64(100000), ptrInt(10), nil, nil, nil, nil,
		nil, nil, nil, false, true, 0,
		time.Now(), time.Now(), nil,
	)

	if err := s.Update(UpdateServiceInput{DiscountPercent: ptrInt(150)}); err == nil {
		t.Fatal("expected a validation error for discountPercent > 100")
	}
}

func TestService_Update_NoOpLeavesUpdatedAtUnchanged(t *testing.T) {
	createdAt := time.Now().Add(-time.Hour)
	s := RehydrateService(
		"id-1", "Title", "slug", nil, nil,
		nil, nil, nil, nil, nil, nil,
		nil, nil, nil, false, true, 0,
		createdAt, createdAt, nil,
	)

	if err := s.Update(UpdateServiceInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.UpdatedAt().Equal(createdAt) {
		t.Errorf("expected updatedAt unchanged on a no-op Update, got %v (was %v)", s.UpdatedAt(), createdAt)
	}
}

func ptrInt64(v int64) *int64 { return &v }
func ptrInt(v int) *int       { return &v }
