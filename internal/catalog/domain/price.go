package domain

import "math"

// NormalizeProductPrice is a 1:1 port of elc-tem's
// modules/catalog/domain/price.ts normalizeProductPrice, applied here at
// write-time (create/update) instead of read-time — see docs/catalog.md for
// why that move is deliberate, not just a port. The three-way branching is
// kept in the exact same order as the TS source (each `case` below maps to
// one `if`/`else if` there); do not reorder or "simplify" it, the order is
// load-bearing (e.g. a row with salePrice > originalPrice and no
// discountPercent falls through all three branches unchanged, faithfully
// reproducing the old TS behavior for that edge case).
func NormalizeProductPrice(originalPriceInput, salePriceInput int64, discountPercentInput float64) (originalPrice, salePrice int64, discountPercent float64) {
	originalPrice = originalPriceInput
	salePrice = salePriceInput
	discountPercent = discountPercentInput

	switch {
	case (salePrice == 0 || salePrice >= originalPrice) && discountPercent > 0 && originalPrice > 0:
		// Case 1: salePrice missing or >= original, but we have a discount percent.
		salePrice = int64(math.Round(float64(originalPrice) * (1 - discountPercent/100)))
	case salePrice > 0 && originalPrice > salePrice:
		// Case 2: a real salePrice lower than original — derive the accurate percent.
		discountPercent = math.Round(((float64(originalPrice) - float64(salePrice)) / float64(originalPrice)) * 100)
	case salePrice == originalPrice || salePrice == 0:
		// Case 3: equal or absent — no discount.
		discountPercent = 0
		salePrice = originalPrice
	}

	return originalPrice, salePrice, discountPercent
}
