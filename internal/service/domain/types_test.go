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
				nil, nil, nil, Seo{}, false, true, 0,
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
		nil, nil, nil, Seo{}, false, true, 0,
		time.Now(), time.Now(), nil,
	)

	if *s.SalePrice() != 90000 {
		t.Fatalf("expected initial sale price 90000, got %d", *s.SalePrice())
	}

	// Simulate an update that only touches discountPercent — the exact case
	// that went stale in the old TS code. UpdatePricing forces both fields
	// to be passed together, so this can't happen here.
	s.UpdatePricing(s.OriginalPrice(), ptrInt(50))

	if *s.SalePrice() != 50000 {
		t.Errorf("expected recomputed sale price 50000, got %d", *s.SalePrice())
	}
}

func ptrInt64(v int64) *int64 { return &v }
func ptrInt(v int) *int       { return &v }
