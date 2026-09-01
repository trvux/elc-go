package main

import (
	"testing"

	"github.com/trvux/elc-go/internal/ai/domain"
)

func TestValidateExtractedPricing(t *testing.T) {
	ptr := func(v float64) *float64 { return &v }

	cases := []struct {
		name    string
		old     domain.Pricing
		new     domain.Pricing
		wantErr bool
	}{
		{
			name:    "first sync, nothing stored yet",
			old:     domain.Pricing{},
			new:     domain.Pricing{InputCacheMissPeak: 0.27, OutputPeak: 1.10},
			wantErr: false,
		},
		{
			name:    "small change within bounds",
			old:     domain.Pricing{InputCacheMissPeak: 0.27, OutputPeak: 1.10},
			new:     domain.Pricing{InputCacheMissPeak: 0.30, OutputPeak: 1.20},
			wantErr: false,
		},
		{
			name:    "negative input price",
			old:     domain.Pricing{},
			new:     domain.Pricing{InputCacheMissPeak: -0.27, OutputPeak: 1.10},
			wantErr: true,
		},
		{
			name:    "negative cache-hit price",
			old:     domain.Pricing{},
			new:     domain.Pricing{InputCacheHitPeak: ptr(-0.1), InputCacheMissPeak: 0.27, OutputPeak: 1.10},
			wantErr: true,
		},
		{
			name:    "zero output price",
			old:     domain.Pricing{},
			new:     domain.Pricing{InputCacheMissPeak: 0.27, OutputPeak: 0},
			wantErr: true,
		},
		{
			name:    "price jumped 1000x vs stored",
			old:     domain.Pricing{InputCacheMissPeak: 0.27, OutputPeak: 1.10},
			new:     domain.Pricing{InputCacheMissPeak: 270, OutputPeak: 1.10},
			wantErr: true,
		},
		{
			name:    "price collapsed to a fraction of stored",
			old:     domain.Pricing{InputCacheMissPeak: 0.27, OutputPeak: 1.10},
			new:     domain.Pricing{InputCacheMissPeak: 0.27, OutputPeak: 0.001},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateExtractedPricing(c.old, c.new)
			if c.wantErr && err == nil {
				t.Errorf("expected an error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}
