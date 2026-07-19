package domain

import (
	"math"
	"sort"
)

// exactValueBucketThreshold: if a number attribute (or price) has this many
// or fewer distinct values across the current result set, show one bucket
// per exact value (Min == Max) instead of grouping into ranges — e.g. gas
// pipe length only ever takes 3 real values (15/20/30m), so "15m"/"20m"/
// "30m" buttons are more honest and useful than an arbitrary range slider.
const exactValueBucketThreshold = 8

// targetRangeBucketCount is how many buckets to aim for when the data is
// genuinely continuous (many distinct values, e.g. price or Wattage).
const targetRangeBucketCount = 5

// BuildNumberBuckets turns a raw slice of matching values into ready-to-click
// suggested ranges, so a shopper who has no idea what unit-scale a spec is
// on gets buttons ("Dưới 9.000", "9.000 - 12.000", ...) instead of a blank
// number input. See AttributeFacet.Buckets / PriceFacet.Buckets.
func BuildNumberBuckets(values []float64) []NumberBucket {
	if len(values) == 0 {
		return nil
	}

	counts := map[float64]int{}
	for _, v := range values {
		counts[v]++
	}
	distinct := make([]float64, 0, len(counts))
	for v := range counts {
		distinct = append(distinct, v)
	}
	sort.Float64s(distinct)

	if len(distinct) <= exactValueBucketThreshold {
		buckets := make([]NumberBucket, len(distinct))
		for i, v := range distinct {
			buckets[i] = NumberBucket{Min: v, Max: v, Count: counts[v]}
		}
		return buckets
	}

	min, max := distinct[0], distinct[len(distinct)-1]
	step := niceNumber((max-min)/float64(targetRangeBucketCount), true)
	if step <= 0 {
		step = 1
	}
	niceMin := math.Floor(min/step) * step

	var buckets []NumberBucket
	for lo := niceMin; lo < max; lo += step {
		hi := lo + step
		count := 0
		for _, v := range values {
			if v >= lo && (v < hi || hi >= max) {
				count++
			}
		}
		if count > 0 {
			buckets = append(buckets, NumberBucket{Min: lo, Max: hi, Count: count})
		}
	}
	return buckets
}

// BuildPriceBuckets generates progressively-widening buckets using the
// classic 1-2-5×10^n "nice number" sequence — the same tiering convention
// Vietnamese e-commerce sites (Thế Giới Di Động, Điện Máy Xanh, Shopee...)
// use for price filters ("Dưới 2 triệu", "2 - 5 triệu", "5 - 10 triệu", "10
// - 20 triệu", ...) rather than equal-width slices — a catalog's prices
// usually span a wide multiplicative range (this catalog: ~6.5tr-67tr for
// aircon alone), so equal-width buckets would dump almost everything into
// one bucket. Fully computed from the actual data, no hardcoded Vietnamese-
// specific price points, so it adapts to any category's real price range
// (a phụ kiện category priced in the hundreds-of-thousands gets its own
// appropriately-scaled tiers, not aircon-sized ones).
func BuildPriceBuckets(values []float64) []NumberBucket {
	if len(values) == 0 {
		return nil
	}
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if min <= 0 {
		min = 1
	}
	if max <= min {
		return []NumberBucket{{Min: min, Max: max, Count: len(values)}}
	}

	boundaries := niceSequence(min, max)
	var buckets []NumberBucket
	for i := 0; i < len(boundaries)-1; i++ {
		lo, hi := boundaries[i], boundaries[i+1]
		isLast := i == len(boundaries)-2
		count := 0
		for _, v := range values {
			if v >= lo && (v < hi || (isLast && v <= hi)) {
				count++
			}
		}
		if count > 0 {
			buckets = append(buckets, NumberBucket{Min: lo, Max: hi, Count: count})
		}
	}
	return buckets
}

// niceSequence returns the ascending 1-2-5×10^n values spanning from the
// largest such value <= min through the smallest such value >= max.
func niceSequence(min, max float64) []float64 {
	mult := [3]float64{1, 2, 5}
	startExp := int(math.Floor(math.Log10(min)))
	endExp := int(math.Floor(math.Log10(max))) + 1

	var seq []float64
	for e := startExp - 1; e <= endExp; e++ {
		for _, m := range mult {
			seq = append(seq, m*math.Pow(10, float64(e)))
		}
	}
	sort.Float64s(seq)

	lo, hi := 0, len(seq)-1
	for i, v := range seq {
		if v <= min {
			lo = i
		}
	}
	for i := len(seq) - 1; i >= 0; i-- {
		if seq[i] >= max {
			hi = i
		}
	}
	if lo >= hi {
		return seq
	}
	return seq[lo : hi+1]
}

// niceNumber rounds x to a "nice" value (1/2/5 × 10^n) — the classic
// Talbot/Lin/Hanrahan "nice numbers for graph labels" algorithm, the same
// technique chart libraries use to pick round axis-tick spacing instead of
// an arbitrary step like 4837.5.
func niceNumber(x float64, round bool) float64 {
	if x <= 0 {
		return 1
	}
	exp := math.Floor(math.Log10(x))
	f := x / math.Pow(10, exp)

	var nf float64
	if round {
		switch {
		case f < 1.5:
			nf = 1
		case f < 3:
			nf = 2
		case f < 7:
			nf = 5
		default:
			nf = 10
		}
	} else {
		switch {
		case f <= 1:
			nf = 1
		case f <= 2:
			nf = 2
		case f <= 5:
			nf = 5
		default:
			nf = 10
		}
	}
	return nf * math.Pow(10, exp)
}
