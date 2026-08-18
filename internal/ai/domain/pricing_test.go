package domain

import (
	"testing"
	"time"

	"math"
)

func deepseekFlashPricing() Pricing {
	hitPeak := 0.044
	return Pricing{
		Currency:           "USD",
		PerMillionTokens:   true,
		InputCacheHitPeak:  &hitPeak,
		InputCacheMissPeak: 1.32,
		OutputPeak:         3.96,
		OffPeakMultiplier:  0.5,
		PeakWindowsUTC:     []PricingWindow{{StartHour: 1, EndHour: 4}, {StartHour: 6, EndHour: 10}},
	}
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestPricing_Cost_Peak(t *testing.T) {
	p := deepseekFlashPricing()
	usage := TokenUsage{InputTokens: 1_000_000, CacheHitTokens: 0, OutputTokens: 1_000_000}
	at := time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC) // inside 01:00-04:00 peak window

	got := p.Cost(usage, at)
	want := 1.32 + 3.96 // 1M miss-tier input + 1M output, no cache hit
	if !almostEqual(got, want) {
		t.Errorf("Cost() = %v, want %v", got, want)
	}
}

func TestPricing_Cost_OffPeak_IsHalfOfPeak(t *testing.T) {
	p := deepseekFlashPricing()
	usage := TokenUsage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	peakAt := time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)     // peak
	offPeakAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) // off-peak

	peakCost := p.Cost(usage, peakAt)
	offPeakCost := p.Cost(usage, offPeakAt)

	if !almostEqual(offPeakCost, peakCost*0.5) {
		t.Errorf("off-peak cost = %v, want half of peak cost %v", offPeakCost, peakCost)
	}
}

func TestPricing_Cost_CacheHitSplitsFromMiss(t *testing.T) {
	p := deepseekFlashPricing()
	usage := TokenUsage{InputTokens: 1_000_000, CacheHitTokens: 400_000, OutputTokens: 0}
	at := time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC) // peak

	got := p.Cost(usage, at)
	want := 400_000.0/1_000_000*0.044 + 600_000.0/1_000_000*1.32
	if !almostEqual(got, want) {
		t.Errorf("Cost() = %v, want %v", got, want)
	}
}

func TestPricing_Cost_NoCacheHitTier_BillsAllAsMiss(t *testing.T) {
	p := Pricing{InputCacheMissPeak: 1.0, OutputPeak: 2.0, OffPeakMultiplier: 1.0}
	usage := TokenUsage{InputTokens: 1_000_000, CacheHitTokens: 900_000, OutputTokens: 0}

	got := p.Cost(usage, time.Now())
	if !almostEqual(got, 1.0) {
		t.Errorf("Cost() = %v, want 1.0 (all input billed at miss rate)", got)
	}
}

func TestPricing_Cost_NoPeakWindows_NeverAppliesDiscount(t *testing.T) {
	p := Pricing{InputCacheMissPeak: 1.0, OutputPeak: 2.0, OffPeakMultiplier: 0.5} // multiplier set but no windows configured
	usage := TokenUsage{InputTokens: 1_000_000, OutputTokens: 0}

	got := p.Cost(usage, time.Now())
	if !almostEqual(got, 1.0) {
		t.Errorf("Cost() = %v, want 1.0 (no peak windows means always full price)", got)
	}
}

func TestPricing_Cost_UnsetMultiplier_DefaultsToOne(t *testing.T) {
	p := Pricing{
		InputCacheMissPeak: 1.0, OutputPeak: 0,
		PeakWindowsUTC: []PricingWindow{{StartHour: 0, EndHour: 1}}, // narrow window so "now" is very likely off-peak
	}
	usage := TokenUsage{InputTokens: 1_000_000}

	got := p.Cost(usage, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)) // definitely off-peak
	if !almostEqual(got, 1.0) {
		t.Errorf("Cost() = %v, want 1.0 (unset multiplier must not silently zero the cost)", got)
	}
}
