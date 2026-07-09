package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// specWhitelist / reverseSpecMapping / getCoolingDirection / getCapacityFromText /
// getGasType / normalizeSpecValue are a 1:1 port of the (until now
// scattered-through-a-500-line-read-time-search-function) HVAC spec
// extraction logic in elc-tem's modules/catalog/application/searchProducts.ts.
// The whole point of this file is that it runs ONCE at write-time
// (create/update — see application/create_product.go, update_product.go),
// not on every search/list request: the result is stored in the
// products.normalized_specs text[] column and matched with a plain array
// overlap (&&) at query time. See docs/catalog.md for the full rationale.
//
// This is a pure function with no I/O — it never queries the DB, never reads
// the product being normalized back from storage, and has no side effects.

// specWhitelist maps a UI-facing facet label to every raw DB spec label
// (Vietnamese, exactly as it appears in products.specs) that should be
// folded into that facet. Copied verbatim from SPEC_WHITELIST in
// searchProducts.ts — do not "clean up" or reformat the Vietnamese strings,
// they must match real spec data byte-for-byte (case/whitespace handled by
// reverseSpecMapping's lookup, not here).
var specWhitelist = map[string][]string{
	"Công suất": {"BTU", "Công suất làm lạnh", "HP", "Ngựa", "Công suất"},
	"Số chiều": {
		"Số chiều",
		"Số chiều làm lạnh",
		"1 chiều/2 chiều",
		"Lạnh/Sưởi",
		"Số chiều hoạt động",
	},
	"Công nghệ": {
		"Inverter",
		"Công nghệ Inverter",
		"Loại Inverter",
		"Tiết kiệm điện",
		"Công nghệ",
		"Công nghệ tiết kiệm điện",
		"Tính năng tiết kiệm điện",
	},
	"Lọc không khí": {
		"Hệ thống lọc khí",
		"Tính năng lọc không khí",
		"Khả năng lọc không khí",
		"Bộ lọc khí",
		"Bộ lọc",
		"Màng lọc",
		"Tính năng lọc bụi",
	},
	"Hiệu suất lọc": {"Hiệu suất lọc không khí", "Hiệu suất lọc"},
	"Loại Gas":      {"Loại Gas", "Môi chất lạnh", "Gas", "Môi chất làm lạnh"},
}

// reverseSpecMapping is built once at init, same as REVERSE_SPEC_MAPPING in
// the TS source: lowercased+trimmed raw label -> UI label.
var reverseSpecMapping = buildReverseSpecMapping()

func buildReverseSpecMapping() map[string]string {
	m := map[string]string{}
	for uiLabel, dbLabels := range specWhitelist {
		for _, dbLabel := range dbLabels {
			m[strings.ToLower(strings.TrimSpace(dbLabel))] = uiLabel
		}
	}
	return m
}

var (
	capacityRegex      = regexp.MustCompile(`(\d+(\.\d+)?)\s*(hp|ngựa|ngua)`)
	leadingNumberRegex = regexp.MustCompile(`^[-+]?\d*\.?\d+`)
	nonDigitDotRegex   = regexp.MustCompile(`[^0-9.]`)
	whitespaceRegex    = regexp.MustCompile(`\s`)
)

// getCoolingDirection ports getCoolingDirection(text) verbatim. strings.ToLower
// works fine on Vietnamese UTF-8 text (no special casing rules needed here,
// same as the TS .toLowerCase()).
func getCoolingDirection(text string) *string {
	l := strings.ToLower(text)
	if strings.Contains(l, "hai chiều") || strings.Contains(l, "2 chiều") || strings.Contains(l, "sưởi") {
		v := "2 Chiều"
		return &v
	}
	if strings.Contains(l, "một chiều") || strings.Contains(l, "1 chiều") ||
		strings.Contains(l, "chỉ làm lạnh") || strings.Contains(l, "lạnh") {
		v := "1 Chiều"
		return &v
	}
	return nil
}

// getCapacityFromText ports getCapacityFromText(text) verbatim: any
// "<number> hp|ngựa|ngua" substring (case-insensitive) becomes "<number> HP".
func getCapacityFromText(text string) *string {
	l := strings.ToLower(text)
	m := capacityRegex.FindStringSubmatch(l)
	if m == nil {
		return nil
	}
	numeric, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil
	}
	v := formatNumber(numeric) + " HP"
	return &v
}

// getGasType ports getGasType(text) verbatim.
func getGasType(text string) *string {
	l := strings.ToLower(text)
	switch {
	case strings.Contains(l, "r32"):
		v := "Gas R32"
		return &v
	case strings.Contains(l, "r410a") || strings.Contains(l, "r410"):
		v := "Gas R410A"
		return &v
	case strings.Contains(l, "r22"):
		v := "Gas R22"
		return &v
	}
	return nil
}

// formatNumber mirrors JS's default Number->string coercion (used by the TS
// template literal `${numeric} HP`) for the plain decimal magnitudes these
// specs ever produce (e.g. 1, 1.5, 2.5) — smallest number of digits that
// round-trips, no trailing ".0".
func formatNumber(n float64) string {
	return strconv.FormatFloat(n, 'f', -1, 64)
}

// normalizeSpecValue ports normalizeSpecValue(uiLabel, rawValue, unit,
// originalLabel) verbatim, INCLUDING one pre-existing bug carried over from
// the TS source on purpose (documented here and in docs/catalog.md, not
// silently fixed): `lower` has all whitespace stripped before the
// "Lọc không khí" branch's substring checks, but three of those checks search
// for a *multi-word* phrase ("bụi mịn", "khử mùi", "tiêu chuẩn", "lọc thô")
// that can never appear in a whitespace-free string. Those three conditions
// are therefore dead in both the original TS and here; only the single-word
// checks (hepa, pm2.5, ion, nanoe, mesh) ever actually match. Confirmed by
// re-reading searchProducts.ts before porting — not a transcription mistake.
func normalizeSpecValue(uiLabel, rawValue string, unit *string, originalLabel string) string {
	val := strings.TrimSpace(rawValue)
	if val == "" {
		return ""
	}

	if unit != nil && *unit != "" && !strings.Contains(strings.ToLower(val), strings.ToLower(*unit)) {
		val = val + *unit
	}

	lower := whitespaceRegex.ReplaceAllString(strings.ToLower(val), "")
	lowerLabel := strings.ToLower(originalLabel)

	switch uiLabel {
	case "Công nghệ":
		if strings.Contains(val, "★") || strings.Contains(lower, "tiếtkiệm") {
			return "Có Inverter"
		}
		isNo := strings.Contains(lower, "không") || strings.Contains(lower, "non-inverter")
		isYes := strings.Contains(lower, "có") || strings.Contains(lower, "inverter")
		if isNo {
			return "Không Inverter"
		}
		if isYes {
			return "Có Inverter"
		}
		return ""

	case "Số chiều":
		if d := getCoolingDirection(val); d != nil {
			return *d
		}
		return ""

	case "Công suất":
		isHP := strings.Contains(lowerLabel, "hp") || strings.Contains(lowerLabel, "ngựa") ||
			strings.Contains(lowerLabel, "ngua") || strings.Contains(lowerLabel, "mã lực") ||
			strings.Contains(lower, "hp") || strings.Contains(lower, "ngựa") || strings.Contains(lower, "ngua")
		if !isHP {
			return ""
		}
		normalized := strings.ReplaceAll(val, ",", ".")
		numeric := parseLeadingFloat(normalized)
		return formatNumber(numeric) + " HP"

	case "Lọc không khí":
		if strings.Contains(lower, "hepa") {
			return "Lọc HEPA"
		}
		if strings.Contains(lower, "pm2.5") || strings.Contains(lower, "bụi mịn") {
			return "Lọc bụi mịn PM2.5"
		}
		if strings.Contains(lower, "ion") || strings.Contains(lower, "khử mùi") || strings.Contains(lower, "nanoe") {
			return "Khử mùi & Diệt khuẩn"
		}
		if (strings.Contains(lower, "tiêu chuẩn") || strings.Contains(lower, "mesh") || strings.Contains(lower, "lọc thô")) &&
			!strings.Contains(lower, "hepa") && !strings.Contains(lower, "pm2.5") {
			return "Lọc bụi tiêu chuẩn"
		}
		return ""

	case "Hiệu suất lọc":
		numStr := nonDigitDotRegex.ReplaceAllString(val, "")
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return ""
		}
		switch {
		case num >= 99.9:
			return "Trên 99.9% (Ultra)"
		case num >= 99:
			return "Trên 99% (HEPA H13)"
		case num >= 95:
			return "Trên 95% (HEPA H11)"
		default:
			return "Lọc tiêu chuẩn"
		}

	case "Loại Gas":
		if g := getGasType(val); g != nil {
			return *g
		}
		return ""
	}

	return val
}

// parseLeadingFloat mimics JS's parseFloat, which parses the leading numeric
// prefix of a string and ignores trailing garbage (e.g. parseFloat("2.5 Hp")
// == 2.5) — strconv.ParseFloat has no equivalent, it fails on trailing
// non-numeric characters, so the leading numeric substring must be extracted
// first.
func parseLeadingFloat(s string) float64 {
	m := leadingNumberRegex.FindString(strings.TrimSpace(s))
	if m == "" {
		return 0
	}
	v, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0
	}
	return v
}

// NormalizeProductSpecs computes the flat "UILabel::Value" facet list stored
// in products.normalized_specs. It combines two sources exactly like the old
// TS searchProducts.ts did when building `availableSpecs`:
//  1. facets derived from the product name itself (cooling direction,
//     capacity, gas type — a name like "Máy lạnh Daikin 1.5HP Inverter"
//     carries real facet data that never appears in a structured spec row);
//  2. facets derived from each whitelisted spec item (or its sub-items).
//
// Unrecognized spec labels (not in specWhitelist) are silently skipped, same
// as the TS `if (!uiLabel) return;` — this is intentional, not an omission:
// only the 6 whitelisted UI facets are meant to be filterable.
func NormalizeProductSpecs(name string, specs []SpecItem) []string {
	result := []string{}
	seen := map[string]struct{}{}
	add := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		result = append(result, s)
	}

	if d := getCoolingDirection(name); d != nil {
		add(fmt.Sprintf("Số chiều::%s", *d))
	}
	if c := getCapacityFromText(name); c != nil {
		add(fmt.Sprintf("Công suất::%s", *c))
	}
	if g := getGasType(name); g != nil {
		add(fmt.Sprintf("Loại Gas::%s", *g))
	}

	for _, item := range specs {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			continue
		}
		uiLabel, ok := reverseSpecMapping[strings.ToLower(label)]
		if !ok {
			continue
		}

		if len(item.Items) > 0 {
			for _, sub := range item.Items {
				if normalized := normalizeSpecValue(uiLabel, sub.Value, sub.Unit, item.Label); normalized != "" {
					add(fmt.Sprintf("%s::%s", uiLabel, normalized))
				}
			}
			continue
		}

		if item.Value != nil {
			if normalized := normalizeSpecValue(uiLabel, *item.Value, item.Unit, item.Label); normalized != "" {
				add(fmt.Sprintf("%s::%s", uiLabel, normalized))
			}
		}
	}

	return result
}
