package domain

import "time"

// PricingWindow is one UTC hour range (start inclusive, end exclusive)
// during which a model's peak price applies.
type PricingWindow struct {
	StartHour int `json:"start_hour"`
	EndHour   int `json:"end_hour"`
}

// Pricing is a model's cost config, stored as JSONB on ai_models and kept
// current by cmd/sync-ai-pricing. Deliberately stores only the peak price
// per tier plus a multiplier for off-peak — not two independent numbers —
// because DeepSeek's own pricing page states off-peak is *exactly* half of
// peak ("Off-peak rates are half of the peak rates"): a formula, not data
// that needs hand-syncing every time peak price changes. A provider with no
// peak/off-peak concept sets OffPeakMultiplier to 1.0 and leaves
// PeakWindowsUTC empty; one with no cache-hit tier leaves
// InputCacheHitPeak nil (all input then bills at InputCacheMissPeak).
type Pricing struct {
	Currency           string          `json:"currency"`
	PerMillionTokens   bool            `json:"per_million_tokens"`
	InputCacheHitPeak  *float64        `json:"input_cache_hit_peak,omitempty"`
	InputCacheMissPeak float64         `json:"input_cache_miss_peak"`
	OutputPeak         float64         `json:"output_peak"`
	OffPeakMultiplier  float64         `json:"off_peak_multiplier"`
	PeakWindowsUTC     []PricingWindow `json:"peak_windows_utc,omitempty"`
}

// TokenUsage is one chat-completion call's token counts, as reported by the
// provider's `usage` field.
type TokenUsage struct {
	InputTokens    int
	OutputTokens   int
	CacheHitTokens int // subset of InputTokens billed at the cache-hit rate
}

// Cost computes what usage cost in USD under this pricing, at the given
// wall-clock time (used to resolve peak vs. off-peak). Pure function, no
// I/O — the whole point of keeping this in domain rather than inline in the
// infrastructure client.
func (p Pricing) Cost(usage TokenUsage, at time.Time) float64 {
	multiplier := 1.0
	if !p.isPeak(at) {
		multiplier = p.offPeakMultiplierOrDefault()
	}

	missTokens := usage.InputTokens
	var hitCost float64
	if p.InputCacheHitPeak != nil {
		hitTokens := usage.CacheHitTokens
		missTokens = usage.InputTokens - hitTokens
		if missTokens < 0 {
			missTokens = 0
		}
		hitCost = float64(hitTokens) / 1_000_000 * (*p.InputCacheHitPeak) * multiplier
	}

	missCost := float64(missTokens) / 1_000_000 * p.InputCacheMissPeak * multiplier
	outputCost := float64(usage.OutputTokens) / 1_000_000 * p.OutputPeak * multiplier
	return hitCost + missCost + outputCost
}

func (p Pricing) offPeakMultiplierOrDefault() float64 {
	if p.OffPeakMultiplier <= 0 {
		// Unset multiplier must never silently zero out the cost — treat
		// it as "no discount" rather than "free".
		return 1.0
	}
	return p.OffPeakMultiplier
}

func (p Pricing) isPeak(at time.Time) bool {
	if len(p.PeakWindowsUTC) == 0 {
		// No windows configured means this provider doesn't distinguish
		// peak/off-peak at all — treat every hour as "peak" so the (unset,
		// defaulting-to-1.0) multiplier never applies.
		return true
	}
	hour := at.UTC().Hour()
	for _, w := range p.PeakWindowsUTC {
		if hour >= w.StartHour && hour < w.EndHour {
			return true
		}
	}
	return false
}
