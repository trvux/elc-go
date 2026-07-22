package application

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// diacriticStripper removes Vietnamese combining diacritical marks after
// NFD decomposition (á -> a + ´, etc.) — this handles every tonal/vowel
// mark, but NOT đ/Đ, which are precomposed Unicode letters in their own
// right rather than "d" + a combining mark, so NFD leaves them untouched.
var diacriticStripper = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// normalizeVietnamese lowercases and strips diacritics (including đ/Đ ->
// d, handled explicitly below since NFD alone doesn't decompose it) for
// fuzzy/typo-tolerant comparison — e.g. matching "đaikin" (a real typo:
// wrong letter, not just a missing accent) against the known brand
// "Daikin". Used by detectBrand and the category-synonym/sub-category
// lookups so they aren't limited to exact accented spelling.
func normalizeVietnamese(s string) string {
	s = strings.ReplaceAll(s, "đ", "d")
	s = strings.ReplaceAll(s, "Đ", "D")
	out, _, err := transform.String(diacriticStripper, s)
	if err != nil {
		return strings.ToLower(s)
	}
	return strings.ToLower(out)
}
