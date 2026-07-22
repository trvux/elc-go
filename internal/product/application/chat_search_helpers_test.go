package application

import "testing"

func TestDetectBrand(t *testing.T) {
	cases := []struct {
		text string
		want string
		ok   bool
	}{
		{"máy lạnh Daikin 1.5HP", "Daikin", true},
		// "đ" is a distinct Vietnamese letter, not decomposable via NFD
		// like an accent mark — this is the exact typo that originally
		// slipped through before normalizeVietnamese handled it explicitly.
		{"máy lặnh đaikin 1.5hp", "Daikin", true},
		{"các mẫu máy lạnh Mitsubishi Heavy nhập khẩu Thái Lan", "Mitsubishi", true},
		{"máy lạnh không rõ hãng", "", false},
		// Fuzzy fallback: a dropped letter ("dakin" missing the second
		// "i") no exact alias regex would match.
		{"mý lạnh dakin", "Daikin", true},
		// Short brand names are excluded from the fuzzy fallback
		// (minFuzzyBrandRunes) — a random unrelated short word shouldn't
		// fuzzy-match "LG".
		{"máy lạnh lá", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			got, ok := detectBrand(tc.text)
			if ok != tc.ok || got != tc.want {
				t.Errorf("detectBrand(%q) = (%q, %v), want (%q, %v)", tc.text, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestDetectSubCategory(t *testing.T) {
	cases := []struct {
		text string
		want string
		ok   bool
	}{
		{"máy lạnh âm trần cassette Daikin", "âm trần", true},
		{"máy lạnh giấu trần nối ống gió Slim Duct", "giấu trần", true},
		{"máy lạnh treo tường 1HP", "treo tường", true},
		{"máy lạnh tủ đứng công suất lớn", "tủ đứng", true},
		{"máy lạnh áp trần cho showroom", "áp trần", true},
		{"máy lạnh 1HP giá rẻ", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			got, ok := detectSubCategory(tc.text)
			if ok != tc.ok || got != tc.want {
				t.Errorf("detectSubCategory(%q) = (%q, %v), want (%q, %v)", tc.text, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestFormatHPTier(t *testing.T) {
	cases := []struct {
		hp   float64
		want string
	}{
		{1, "1 HP"},
		{2, "2 HP"},
		{1.5, "1.5 HP"},
		{2.5, "2.5 HP"},
	}
	for _, tc := range cases {
		if got := formatHPTier(tc.hp); got != tc.want {
			t.Errorf("formatHPTier(%v) = %q, want %q", tc.hp, got, tc.want)
		}
	}
}

func TestDetectGasType(t *testing.T) {
	cases := []struct {
		text string
		want string
		ok   bool
	}{
		{"máy lạnh dùng gas R32 tiết kiệm điện", "R32", true},
		{"điều hòa gas R410A", "R410A", true},
		{"máy lạnh không rõ loại gas", "", false},
	}
	for _, tc := range cases {
		got, ok := detectGasType(tc.text)
		if ok != tc.ok || got != tc.want {
			t.Errorf("detectGasType(%q) = (%q, %v), want (%q, %v)", tc.text, got, ok, tc.want, tc.ok)
		}
	}
}
