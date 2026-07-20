package seo

import "testing"

func strPtr(s string) *string { return &s }

func TestUnchanged(t *testing.T) {
	cases := []struct {
		name     string
		current  *string
		newValue *string
		wantSame bool
	}{
		{"both nil", nil, nil, true},
		{"same value", strPtr("abc"), strPtr("abc"), true},
		{"different value", strPtr("abc"), strPtr("xyz"), false},
		{"nil to value", nil, strPtr("abc"), false},
		{"value to nil", strPtr("abc"), nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Unchanged(c.current, c.newValue); got != c.wantSame {
				t.Errorf("Unchanged(%v, %v) = %v, want %v", c.current, c.newValue, got, c.wantSame)
			}
		})
	}
}
