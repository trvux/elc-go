package infrastructure

import "testing"

func TestCapacityFamilyKey(t *testing.T) {
	cases := []struct {
		name string
		mpn  string
		want string
	}{
		{"daikin wall-mount", "FTKY25ZVMV", "FTKYZVMV"},
		{"daikin wall-mount other capacity, same model", "FTKY35ZVMV", "FTKYZVMV"},
		{"different remote/feature variant, not a capacity sibling", "FTKM25AVMV", "FTKMAVMV"},
		{"menred fresh-air, no digits stripped from suffix dot", "NET.1000", "NET."},
		{"lower-case input normalizes", "ftky25zvmv", "FTKYZVMV"},
		{"no digits at all", "FTKY", "FTKY"},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := capacityFamilyKey(c.mpn); got != c.want {
				t.Errorf("capacityFamilyKey(%q) = %q, want %q", c.mpn, got, c.want)
			}
		})
	}
}
