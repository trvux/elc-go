package domain

import "testing"

func TestNormalizeProductPrice(t *testing.T) {
	tests := []struct {
		name                string
		originalPrice       int64
		salePrice           int64
		discountPercent     float64
		wantOriginal        int64
		wantSale            int64
		wantDiscountPercent float64
	}{
		{
			name:          "discount percent given, no sale price -> derives sale price",
			originalPrice: 100000, salePrice: 0, discountPercent: 10,
			wantOriginal: 100000, wantSale: 90000, wantDiscountPercent: 10,
		},
		{
			name:          "sale price lower than original -> derives discount percent",
			originalPrice: 100000, salePrice: 75000, discountPercent: 0,
			wantOriginal: 100000, wantSale: 75000, wantDiscountPercent: 25,
		},
		{
			// salePrice >= originalPrice AND discountPercent > 0 hits case 1
			// first (recompute salePrice from the percent) — case 3 ("equal ->
			// force discount to 0") only wins when discountPercent is 0, see
			// the next case below.
			name:          "sale price equals original but discount percent given -> recomputes sale price",
			originalPrice: 100000, salePrice: 100000, discountPercent: 15,
			wantOriginal: 100000, wantSale: 85000, wantDiscountPercent: 15,
		},
		{
			name:          "sale price equals original, no discount -> discount forced to 0",
			originalPrice: 100000, salePrice: 100000, discountPercent: 0,
			wantOriginal: 100000, wantSale: 100000, wantDiscountPercent: 0,
		},
		{
			name:          "no sale price, no discount -> sale price becomes original",
			originalPrice: 100000, salePrice: 0, discountPercent: 0,
			wantOriginal: 100000, wantSale: 100000, wantDiscountPercent: 0,
		},
		{
			name:          "everything zero -> stays zero",
			originalPrice: 0, salePrice: 0, discountPercent: 0,
			wantOriginal: 0, wantSale: 0, wantDiscountPercent: 0,
		},
		{
			name:          "sale price greater than original with no discount -> left untouched (ported edge case)",
			originalPrice: 100000, salePrice: 150000, discountPercent: 0,
			wantOriginal: 100000, wantSale: 150000, wantDiscountPercent: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOriginal, gotSale, gotDiscount := NormalizeProductPrice(tt.originalPrice, tt.salePrice, tt.discountPercent)
			if gotOriginal != tt.wantOriginal {
				t.Errorf("originalPrice = %d, want %d", gotOriginal, tt.wantOriginal)
			}
			if gotSale != tt.wantSale {
				t.Errorf("salePrice = %d, want %d", gotSale, tt.wantSale)
			}
			if gotDiscount != tt.wantDiscountPercent {
				t.Errorf("discountPercent = %v, want %v", gotDiscount, tt.wantDiscountPercent)
			}
		})
	}
}
