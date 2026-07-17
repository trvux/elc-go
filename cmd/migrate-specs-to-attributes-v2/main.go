// One-off migration (v2) — supersedes cmd/migrate-specs-to-attributes for
// the redesign's data-migration epic. Reads raw products.specs from a
// SOURCE database (old flat schema, e.g. the production backup restored
// into elc_prod_ref) and writes structured product_attribute_values into a
// TARGET database (new schema, e.g. local dev `elc`) — cross-database by
// design, since the redesign's new attribute_definitions/
// category_attribute_definitions live in the target, not the source.
//
// Adds three things v1 deliberately skipped as too ambiguous, now handled
// with specific, narrow, non-guessing rules (see comments below):
//  1. Multi-unit same-value groups (Items[] holding the same capacity in
//     HP/kW/BTU, unlabeled positional sub-items) — picks the sub-item whose
//     text contains the target definition's own unit.
//  2. Section-header state tracking ("DÀN LẠNH"/"THÔNG TIN DÀN LẠNH" etc.,
//     a flat {label,value} entry whose value is actually a model code) —
//     captures the model code into ma_dan_lanh/ma_dan_nong/ma_mat_na, and
//     routes ONLY the exact bare labels "Kích thước"/"Trọng lượng"/"Độ ồn"
//     appearing while a section is active to that section's specific
//     definition (kich_thuoc_dan_lanh vs kich_thuoc_dan_nong, etc.) — any
//     other nearby label (e.g. "Kích thước Máy lạnh") is NOT touched by
//     this rule, still too ambiguous to route safely.
//  3. Generic nested groups with no recognized pattern (e.g. "Nguồn điện"
//     -> ["220V","50Hz"], "Bộ lọc khí" -> [feature, feature, ...]) — joined
//     with ", " into one text value under a definition matching the
//     group's own label, IF such a definition already exists (never
//     auto-creates new definitions — those were curated by hand from a
//     real label audit, see scripts/seed-attribute-definitions-*.sql).
//
// Same conservative rules as v1 otherwise: exact/synonym label match only,
// number parsing rejects anything not a clean decimal, select requires an
// exact (punctuation-folded) option match, idempotent upsert.
//
// Usage:
//
//	SOURCE_DATABASE_URL=... TARGET_DATABASE_URL=... go run ./cmd/migrate-specs-to-attributes-v2
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/unicode/norm"
)

type specSubItem struct {
	Label string  `json:"label"`
	Value string  `json:"value"`
	Unit  *string `json:"unit,omitempty"`
}

type specItem struct {
	Label string        `json:"label"`
	Value *string       `json:"value,omitempty"`
	Unit  *string       `json:"unit,omitempty"`
	Items []specSubItem `json:"items,omitempty"`
}

type attributeDef struct {
	id       string
	code     string
	name     string
	dataType string
	unit     string
	options  []string
}

var acSynonyms = map[string]string{
	"loại máy":               "loai_may",
	"công nghệ inverter":     "cong_nghe_inverter",
	"loại gas lạnh":          "loai_gas_lanh",
	"môi chất lạnh":          "loai_gas_lanh",
	"môi chất làm lạnh":      "loai_gas_lanh",
	"môi chất lạnh (gas)":    "loai_gas_lanh",
	"nguồn điện":             "nguon_dien",
	"nguồn điện (ph/v/hz)":   "nguon_dien",
	"điện áp vào":            "nguon_dien",
	"xuất xứ":                "xuat_xu",
	"sản xuất tại":           "xuat_xu",
	"nguồn gốc":              "xuat_xu",
	"độ ồn":                  "do_on",
	"độ ồn (cao / cực thấp)": "do_on",
	"độ ồn (cao / trung bình / thấp / yên tĩnh)":  "do_on",
	"độ ồn (cao / trung bình / thấp)":             "do_on",
	"độ ồn (cao/trung bình/thấp)":                 "do_on",
	"kích thước dàn lạnh":                         "kich_thuoc_dan_lanh",
	"kích thước dàn lạnh (rxcxs)":                 "kich_thuoc_dan_lanh",
	"khối lượng dàn lạnh":                         "trong_luong_dan_lanh",
	"trọng lượng dàn lạnh":                        "trong_luong_dan_lanh",
	"kích thước dàn nóng":                         "kich_thuoc_dan_nong",
	"kích thước dàn nóng (rxcxs)":                 "kich_thuoc_dan_nong",
	"khối lượng dàn nóng":                         "trong_luong_dan_nong",
	"trọng lượng dàn nóng":                        "trong_luong_dan_nong",
	"chiều dài tối đa":                            "chieu_dai_ong_gas_toi_da",
	"chiều dài ống gas tối đa":                    "chieu_dai_ong_gas_toi_da",
	"chiều dài ống gas tối đa (m)":                "chieu_dai_ong_gas_toi_da",
	"chiều dài đường ống":                         "chieu_dai_ong_gas_toi_da",
	"chiều dài đường ống tối đa":                  "chieu_dai_ong_gas_toi_da",
	"chiều dài tối đa ống nối các thiết bị":       "chieu_dai_ong_gas_toi_da",
	"chênh lệch độ cao tối đa":                    "chenh_lech_do_cao_toi_da",
	"chênh lệch độ cao (tối đa)":                  "chenh_lech_do_cao_toi_da",
	"chênh lệch độ cao (tối đa) (m)":              "chenh_lech_do_cao_toi_da",
	"chênh lệch độ cao tối đa giữa dàn nóng-lạnh": "chenh_lech_do_cao_toi_da",
	"chiều cao lắp đặt tối đa giữa dàn nóng-lạnh": "chenh_lech_do_cao_toi_da",
	"chênh lệch độ cao tối đa trong lắp đặt":      "chenh_lech_do_cao_toi_da",
	"độ cao chênh lệch tối đa":                    "chenh_lech_do_cao_toi_da",
	"hiệu suất năng lượng (cspf)":                 "hieu_suat_cspf",
	"hiệu suất năng lượng cspf":                   "hieu_suat_cspf",
	"chỉ số hiệu suất năng lượng (cspf)":          "hieu_suat_cspf",
	"cspf":                           "hieu_suat_cspf",
	"hiệu số cspf":                   "hieu_suat_cspf",
	"phạm vi làm lạnh hiệu quả":      "pham_vi_lam_lanh",
	"sử dụng cho phòng":              "pham_vi_lam_lanh",
	"phạm vi làm lạnh":               "pham_vi_lam_lanh",
	"điện năng tiêu thụ":             "dien_nang_tieu_thu",
	"điện năng tiêu thụ định mức":    "dien_nang_tieu_thu",
	"công suất tiêu thụ":             "dien_nang_tieu_thu",
	"công suất tiêu thụ điện":        "dien_nang_tieu_thu",
	"công suất tiêu thụ trung bình":  "dien_nang_tieu_thu",
	"nhãn năng lượng tiết kiệm điện": "nhan_nang_luong",
	"tiết kiệm điện":                 "nhan_nang_luong",
}

// menredSynonyms folds real label variants (found via the 2026-07-17 audit)
// onto the original 6-field Menred seed set instead of creating near-dupe
// definitions for "Kích thước máy" vs "Kích thước" etc.
var menredSynonyms = map[string]string{
	"kích thước máy":          "kich_thuoc",
	"kích thước máy (mm):  ;": "kich_thuoc",
	"hiệu suất lọc không khí": "hieu_suat_loc",
	"hiệu quả làm mát":        "hieu_suat_loc",
	"hiệu quả sưởi ấm":        "hieu_suat_loc",
}

// sectionHeaders recognizes a flat {label,value} entry whose VALUE is
// actually a model code marking the start of a new section — value is
// never parsed as a spec, only routed to captureCode (if the definition
// exists) and used to set the current section for subsequent generic
// labels. "Thông tin chung" resets to no section without capturing
// anything (its value is just the two codes concatenated, redundant with
// the two captured separately).
var sectionHeaders = map[string]struct{ section, captureCode string }{
	"dàn lạnh":           {"dan_lanh", "ma_dan_lanh"},
	"thông tin dàn lạnh": {"dan_lanh", "ma_dan_lanh"},
	"dàn lạnh model":     {"dan_lanh", "ma_dan_lanh"},
	"dàn nóng":           {"dan_nong", "ma_dan_nong"},
	"thông tin dàn nóng": {"dan_nong", "ma_dan_nong"},
	"mặt nạ":             {"mat_na", "ma_mat_na"},
	"thông tin mặt nạ":   {"mat_na", "ma_mat_na"},
	"thông tin chung":    {"", ""},
}

// sectionRoutedLabels: labels re-routed while a section is active, matched
// by PREFIX (see sectionRoutePrefixes below) — verified safe against real
// data (2026-07-17): across every AC sub-category sampled, "Kích thước
// Điều hoà"/"Kích thước Máy lạnh"/"Kích thước dàn lạnh"/"Kích thước (Cao x
// Rộng x Dày)" etc. are ALL just inconsistent wording for "kích thước của
// khối vừa được nêu tên ở section header phía trên" — staff used "Điều
// hoà"/"Máy lạnh" as interchangeable (and, confusingly, not literally
// meaningful) words for "dàn lạnh"/"dàn nóng" depending on which section
// they were filling in, not as a distinct third concept. The section
// header immediately above is the reliable signal, not the suffix wording.
var sectionRoutedLabels = map[string]map[string]string{
	"dan_lanh": {"kích thước": "kich_thuoc_dan_lanh", "trọng lượng": "trong_luong_dan_lanh", "khối lượng": "trong_luong_dan_lanh", "độ ồn": "do_on_dan_lanh"},
	"dan_nong": {"kích thước": "kich_thuoc_dan_nong", "trọng lượng": "trong_luong_dan_nong", "khối lượng": "trong_luong_dan_nong", "độ ồn": "do_on_dan_nong"},
	"mat_na":   {"kích thước": "kich_thuoc_mat_na", "trọng lượng": "trong_luong_mat_na", "khối lượng": "trong_luong_mat_na"},
}

// sectionRoutePrefixes are matched against the START of a normalized label
// (not exact-equality) — e.g. "kích thước điều hoà" and "kích thước (cao x
// rộng x dày)" both start with "kích thước".
var sectionRoutePrefixes = []string{"kích thước", "trọng lượng", "khối lượng", "độ ồn"}

// matchSectionRoute finds the longest sectionRoutePrefixes entry that norm
// starts with, and returns its routed code for the given section (if any).
func matchSectionRoute(section, norm string) (string, bool) {
	routes, ok := sectionRoutedLabels[section]
	if !ok {
		return "", false
	}
	best := ""
	for _, prefix := range sectionRoutePrefixes {
		if strings.HasPrefix(norm, prefix) && len(prefix) > len(best) {
			best = prefix
		}
	}
	if best == "" {
		return "", false
	}
	code, ok := routes[best]
	return code, ok
}

// normalizeLabel prepares a label for comparison — critically including
// Unicode NFC normalization (2026-07-17 finding): the old specs jsonb has
// Vietnamese text entered inconsistently, some diacritics as a single
// precomposed codepoint ("ấ" = U+1EA5) and others as a base letter plus a
// combining accent ("â" U+00E2 + combining acute U+0301) — visually
// identical, byte-different, so a plain string compare silently failed to
// match real labels against attribute_definitions.name/acSynonyms. Without
// this, `normalizeLabel(d.name) == norm` and every synonym-map lookup
// below would randomly miss matches depending on which normalization form
// a given product's data happened to use.
func normalizeLabel(s string) string {
	s = norm.NFC.String(s)
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.ToLower(s)
}

func punctFold(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r == ' ' || r == '-' || r == '_' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func stripOwnUnit(value, unit string) string {
	if unit == "" {
		return value
	}
	v := strings.TrimSpace(value)
	for _, suf := range []string{"(" + unit + ")", unit} {
		if strings.HasSuffix(strings.ToLower(v), strings.ToLower(suf)) {
			trimmed := strings.TrimSpace(v[:len(v)-len(suf)])
			trimmed = strings.TrimSuffix(trimmed, "(")
			trimmed = strings.TrimSpace(trimmed)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return value
}

// capacityNumberFromMultiUnit extracts the leading number from a capacity
// sub-item value like "11.100 Btu/h" or "24,200 (4,100 -25,600)" — real
// aircon spec data (2026-07-17 audit) expresses the same [HP, kW, BTU/h]
// capacity group two ways: a clean "<number> <unit>" triple, or a rated
// value followed by a parenthetical min-max range with no unit text at
// all, with "," and "." both used inconsistently as thousands separators
// (never as a decimal point — BTU capacities are always whole numbers in
// the thousands, unlike the HP/kW entries in the same group). Cuts at the
// first "(", strips all grouping punctuation, and requires the result to
// be a plausible capacity magnitude before trusting it as a whole number
// rather than a genuine small decimal (e.g. a kW value read by mistake).
func capacityNumberFromMultiUnit(raw string) (float64, bool) {
	s := strings.TrimSpace(raw)
	if i := strings.IndexByte(s, '('); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	end := 0
	for end < len(s) && (s[end] >= '0' && s[end] <= '9' || s[end] == ',' || s[end] == '.') {
		end++
	}
	numPart := strings.TrimRight(s[:end], ",.")
	if numPart == "" {
		return 0, false
	}
	stripped := strings.NewReplacer(",", "", ".", "").Replace(numPart)
	n, err := strconv.ParseFloat(stripped, 64)
	if err != nil {
		return 0, false
	}
	if n < 100 {
		return 0, false
	}
	return n, true
}

// hpOptionFromMultiUnit extracts a plain HP number (e.g. "1.5" from
// "1.5 HP (1.5 Ngựa)" or "2 ) ") and formats it to match this business's
// "<N> HP" select options — HP values in this group are always the first
// sub-item and never use thousands separators (unlike BTU), so this is
// simpler than capacityNumberFromMultiUnit.
func hpOptionFromMultiUnit(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	end := 0
	for end < len(s) && (s[end] >= '0' && s[end] <= '9' || s[end] == '.') {
		end++
	}
	numPart := strings.Trim(s[:end], ".")
	if numPart == "" {
		return "", false
	}
	n, err := strconv.ParseFloat(numPart, 64)
	if err != nil || n <= 0 {
		return "", false
	}
	if n == float64(int64(n)) {
		return fmt.Sprintf("%d HP", int64(n)), true
	}
	return fmt.Sprintf("%g HP", n), true
}

// pendingValue is what we've decided to write, before the final DB write.
type pendingValue struct {
	defID string
	text  *string
	num   *float64
	boo   *bool
}

// leadingNumber extracts a usable numeric value from a raw spec value that
// may carry a comparison prefix ("<=12"), a parenthetical min-max range
// ("930 (120 ~ 1,100)"), and/or thousands commas — found 2026-07-18 sampling
// products whose sub-items separate value from unit (see flatLikeValue):
// weight/length/power/area fields in this data consistently use comma
// (never dot, unlike BTU — see capacityNumberFromMultiUnit) as the
// thousands separator, and a parenthetical suffix is always a supplementary
// min-max range, never the primary reading. Taking the leading/rated figure
// before it is the same judgment call already validated for BTU, applied
// consistently rather than case-by-case.
func leadingNumber(raw string) (float64, bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimLeft(s, "<>=~≈ \t")
	if i := strings.IndexByte(s, '('); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func setValueForDataType(target attributeDef, rawValue string) (pendingValue, bool) {
	return setValueForDataTypeWithUnit(target, rawValue, "")
}

// setValueForDataTypeWithUnit is setValueForDataType plus an optional
// subUnit — when a spec sub-item carries its unit as a separate JSON field
// rather than embedded in the value text (see flatLikeValue's doc
// comment), there is nothing to strip from the value string, so
// stripOwnUnit is skipped entirely rather than risking it eating part of a
// genuine number.
func setValueForDataTypeWithUnit(target attributeDef, rawValue, subUnit string) (pendingValue, bool) {
	value := strings.TrimSpace(rawValue)
	if value == "" {
		return pendingValue{}, false
	}
	switch target.dataType {
	case "text":
		v := value
		return pendingValue{defID: target.id, text: &v}, true
	case "select":
		normValue := punctFold(value)
		for _, opt := range target.options {
			if punctFold(opt) == normValue {
				o := opt
				return pendingValue{defID: target.id, text: &o}, true
			}
		}
		return pendingValue{}, false
	case "number":
		v := value
		if subUnit == "" {
			v = stripOwnUnit(value, target.unit)
		}
		n, ok := leadingNumber(v)
		if !ok {
			return pendingValue{}, false
		}
		return pendingValue{defID: target.id, num: &n}, true
	case "boolean":
		lower := strings.ToLower(value)
		lower = strings.TrimSuffix(strings.TrimSpace(lower), ".")
		var b bool
		switch lower {
		case "có":
			b = true
		case "không":
			b = false
		default:
			return pendingValue{}, false
		}
		return pendingValue{defID: target.id, boo: &b}, true
	}
	return pendingValue{}, false
}

// flatLikeValue unwraps an item into an equivalent flat (value, unit) pair
// if it's already flat, OR if it's a nested group with exactly one
// blank-labeled sub-item — a data-entry style (found 2026-07-18) where even
// single scalar values get wrapped in items[] with the unit on the
// sub-item rather than the parent (e.g. "Trọng lượng" ->
// items:[{unit:"kg", value:"9"}]). Returns ok=false for genuine multi-part
// groups (HP/kW/BTU triples, feature lists, etc.), which stay on the
// specialized Rule 1/Rule 3 path below.
func flatLikeValue(item specItem) (value string, unit string, ok bool) {
	if item.Items == nil {
		if item.Value == nil {
			return "", "", false
		}
		u := ""
		if item.Unit != nil {
			u = *item.Unit
		}
		return *item.Value, u, true
	}
	if len(item.Items) == 1 && strings.TrimSpace(item.Items[0].Label) == "" {
		sub := item.Items[0]
		u := ""
		if sub.Unit != nil {
			u = *sub.Unit
		}
		return sub.Value, u, true
	}
	return "", "", false
}

func main() {
	ctx := context.Background()
	src, err := pgxpool.New(ctx, os.Getenv("SOURCE_DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect source: %v", err)
	}
	defer src.Close()
	dst, err := pgxpool.New(ctx, os.Getenv("TARGET_DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect target: %v", err)
	}
	defer dst.Close()

	// Load target definitions, keyed by code (global lookup) and grouped by
	// category (via category_attribute_definitions; zero rows = global).
	byCode := map[string]attributeDef{}
	drows, err := dst.Query(ctx, `SELECT id, code, name, data_type, unit, options FROM attribute_definitions WHERE deleted_at IS NULL`)
	if err != nil {
		log.Fatalf("load target definitions: %v", err)
	}
	for drows.Next() {
		var d attributeDef
		var unit *string
		if err := drows.Scan(&d.id, &d.code, &d.name, &d.dataType, &unit, &d.options); err != nil {
			log.Fatalf("scan target definition: %v", err)
		}
		if unit != nil {
			d.unit = *unit
		}
		byCode[d.code] = d
	}
	drows.Close()
	log.Printf("target: %d attribute_definitions loaded", len(byCode))

	defsByCategory := map[string][]attributeDef{}
	var globalDefs []attributeDef
	catRows, err := dst.Query(ctx, `
		SELECT ad.id, ad.code, ad.name, ad.data_type, ad.unit, ad.options, cad.category_id
		FROM attribute_definitions ad
		LEFT JOIN category_attribute_definitions cad ON cad.attribute_definition_id = ad.id
		WHERE ad.deleted_at IS NULL`)
	if err != nil {
		log.Fatalf("load category attachments: %v", err)
	}
	attached := map[string]bool{}
	for catRows.Next() {
		var d attributeDef
		var unit *string
		var categoryID *string
		if err := catRows.Scan(&d.id, &d.code, &d.name, &d.dataType, &unit, &d.options, &categoryID); err != nil {
			log.Fatalf("scan category attachment: %v", err)
		}
		if unit != nil {
			d.unit = *unit
		}
		if categoryID != nil {
			defsByCategory[*categoryID] = append(defsByCategory[*categoryID], d)
			attached[d.id] = true
		}
	}
	catRows.Close()
	for _, d := range byCode {
		if !attached[d.id] {
			globalDefs = append(globalDefs, d)
		}
	}

	acCategoryIDs := map[string]bool{}
	acRows, err := dst.Query(ctx, `SELECT id FROM categories WHERE name IN (
		'Máy lạnh treo tường','Máy lạnh âm trần đa hướng thổi','Máy lạnh áp trần',
		'Máy lạnh giấu trần nối ống gió','Máy lạnh tủ đứng') AND deleted_at IS NULL`)
	if err != nil {
		log.Fatalf("load ac categories: %v", err)
	}
	for acRows.Next() {
		var id string
		if err := acRows.Scan(&id); err != nil {
			log.Fatalf("scan ac category: %v", err)
		}
		acCategoryIDs[id] = true
	}
	acRows.Close()

	menredCategoryID := ""
	if err := dst.QueryRow(ctx, `SELECT id FROM categories WHERE name = 'Máy cấp khí tươi, lọc không khí' AND deleted_at IS NULL`).Scan(&menredCategoryID); err != nil {
		log.Printf("WARNING: could not load Menred category id: %v", err)
	}

	// Load products from SOURCE (raw specs) — product/category ids are
	// identical between source and target (same origin snapshot), verified
	// manually before running this script.
	type productRow struct {
		id, categoryID string
		specs          []byte
	}
	var products []productRow
	prows, err := src.Query(ctx, `SELECT id, category_id, specs FROM products WHERE deleted_at IS NULL`)
	if err != nil {
		log.Fatalf("load source products: %v", err)
	}
	for prows.Next() {
		var p productRow
		if err := prows.Scan(&p.id, &p.categoryID, &p.specs); err != nil {
			log.Fatalf("scan source product: %v", err)
		}
		products = append(products, p)
	}
	prows.Close()
	log.Printf("source: %d products", len(products))

	totalEntries, matched, skipped := 0, 0, 0

	for _, p := range products {
		var specs []specItem
		if len(p.specs) == 0 {
			continue
		}
		if err := json.Unmarshal(p.specs, &specs); err != nil {
			continue
		}

		candidates := append(append([]attributeDef{}, defsByCategory[p.categoryID]...), globalDefs...)
		isAC := acCategoryIDs[p.categoryID]
		isMenred := p.categoryID == menredCategoryID
		byCodeCandidate := func(code string) (attributeDef, bool) {
			for _, d := range candidates {
				if d.code == code {
					return d, true
				}
			}
			return attributeDef{}, false
		}

		pending := map[string]pendingValue{} // defID -> value, last-write-wins within one product
		section := ""

		for _, item := range specs {
			norm := normalizeLabel(item.Label)
			if norm == "" {
				continue
			}

			// Rule 2: section header (always genuinely flat in every
			// sample seen — never single-item-wrapped).
			if item.Items == nil && item.Value != nil {
				if hdr, ok := sectionHeaders[norm]; ok {
					section = hdr.section
					if hdr.captureCode != "" {
						if def, ok := byCodeCandidate(hdr.captureCode); ok {
							if pv, ok := setValueForDataType(def, *item.Value); ok {
								pending[def.id] = pv
								matched++
							}
						}
					}
					continue
				}
			}

			// Flat-like path: covers both genuinely flat {label,value}
			// entries and single-blank-sub-item nested groups (see
			// flatLikeValue doc comment) through one unified pipeline —
			// section-routing, acSynonym/menredSynonym, then name match.
			if fv, funit, ok := flatLikeValue(item); ok {
				totalEntries++
				value := strings.TrimSpace(fv)
				if value == "" {
					continue
				}

				if section != "" {
					if code, ok := matchSectionRoute(section, norm); ok {
						if def, ok := byCodeCandidate(code); ok {
							if pv, ok := setValueForDataTypeWithUnit(def, value, funit); ok {
								pending[def.id] = pv
								matched++
								continue
							}
						}
					}
				}

				var target *attributeDef
				if code, ok := acSynonymOrDirect(norm, isAC); ok {
					if d, ok := byCodeCandidate(code); ok {
						target = &d
					}
				}
				if target == nil && isMenred {
					if code, ok := menredSynonyms[norm]; ok {
						if d, ok := byCodeCandidate(code); ok {
							target = &d
						}
					}
				}
				if target == nil {
					for _, d := range candidates {
						if normalizeLabel(d.name) == norm {
							target = &d
							break
						}
					}
				}
				if target == nil {
					skipped++
					continue
				}
				if pv, ok := setValueForDataTypeWithUnit(*target, value, funit); ok {
					pending[target.id] = pv
					matched++
				} else {
					skipped++
				}
				continue
			}

			// Rule 1: multi-unit same-value group (unlabeled sub-items,
			// more than one — the single-sub-item case above already
			// handled by flatLikeValue).
			if item.Items != nil {
				totalEntries++
				allBlank := true
				for _, sub := range item.Items {
					if strings.TrimSpace(sub.Label) != "" {
						allBlank = false
						break
					}
				}
				var target *attributeDef
				if code, ok := acSynonymOrDirect(norm, isAC); ok {
					if d, ok := byCodeCandidate(code); ok {
						target = &d
					}
				}
				if target == nil {
					for _, d := range candidates {
						if normalizeLabel(d.name) == norm {
							target = &d
							break
						}
					}
				}
				if allBlank && target != nil && target.dataType == "number" && target.unit != "" {
					isBTUField := target.code == "cong_suat_lam_lanh_btu" || target.code == "cong_suat_suoi_btu"
					unitLower := strings.ToLower(target.unit)
					found := false
					for _, sub := range item.Items {
						if strings.Contains(strings.ToLower(sub.Value), unitLower) {
							// BTU fields always use the thousands-separator-
							// aware parser (see capacityNumberFromMultiUnit
							// doc comment) — a plain strconv.ParseFloat on
							// "11.100 Btu/h" silently misreads the Vietnamese
							// thousands dot as a decimal point (-> 11.1).
							if isBTUField {
								if n, ok := capacityNumberFromMultiUnit(sub.Value); ok {
									pending[target.id] = pendingValue{defID: target.id, num: &n}
									matched++
									found = true
								}
							} else if pv, ok := setValueForDataType(*target, sub.Value); ok {
								pending[target.id] = pv
								matched++
								found = true
							}
							break
						}
					}
					// Positional fallback, BTU fields only: real data
					// consistently orders this group [HP, kW, BTU/h] even
					// when the BTU sub-item carries no unit text at all.
					if !found && len(item.Items) == 3 && isBTUField {
						if n, ok := capacityNumberFromMultiUnit(item.Items[2].Value); ok {
							pending[target.id] = pendingValue{defID: target.id, num: &n}
							matched++
							found = true
						}
					}
					// Positional fallback for the matching select-type HP field
					// (phan_khuc_hp) — same [HP, kW, BTU/h] group, index 0.
					// Gated on isBTUField + exactly 3 items so this never
					// misfires against an unrelated 2-item number group
					// (e.g. "Sử dụng cho phòng" -> [m², m³]).
					if isBTUField && len(item.Items) == 3 {
						if hpDef, ok := byCodeCandidate("phan_khuc_hp"); ok {
							if opt, ok := hpOptionFromMultiUnit(item.Items[0].Value); ok {
								if pv, ok := setValueForDataType(hpDef, opt); ok {
									pending[hpDef.id] = pv
									matched++
								}
							}
						}
					}
					if !found {
						skipped++
					}
					continue
				}
				// Rule 3: generic nested group — join sub-values as text,
				// only if a definition for the group's own label exists.
				if target != nil {
					var parts []string
					for _, sub := range item.Items {
						v := strings.TrimSpace(sub.Value)
						if v != "" {
							parts = append(parts, v)
						}
					}
					if len(parts) > 0 {
						joined := strings.Join(parts, ", ")
						if pv, ok := setValueForDataType(*target, joined); ok {
							pending[target.id] = pv
							matched++
							continue
						}
					}
				}
				skipped++
				continue
			}
		}

		for defID, pv := range pending {
			// Safety guard: never persist a value with nothing in it — a
			// pendingValue should only ever exist via setValueForDataType's
			// ok=true branch, but this is a cheap, worthwhile backstop
			// against any future code path that forgets the check.
			if pv.text == nil && pv.num == nil && pv.boo == nil {
				log.Printf("WARNING: skipping empty pending value for product=%s def=%s (should not happen)", p.id, defID)
				continue
			}
			if _, err := dst.Exec(ctx, `
				INSERT INTO product_attribute_values (product_id, attribute_definition_id, value_text, value_number, value_boolean)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (product_id, attribute_definition_id) WHERE deleted_at IS NULL
				DO UPDATE SET value_text = $3, value_number = $4, value_boolean = $5, updated_at = now()`,
				p.id, pv.defID, pv.text, pv.num, pv.boo,
			); err != nil {
				log.Fatalf("write value product=%s def=%s: %v", p.id, pv.defID, err)
			}
		}
	}

	fmt.Printf("Spec entries examined: %d\nMatched -> product_attribute_values: %d\nLeft unmapped (still in products.specs): %d\n", totalEntries, matched, skipped)
}

func acSynonymOrDirect(norm string, isAC bool) (string, bool) {
	if !isAC {
		return "", false
	}
	code, ok := acSynonyms[norm]
	return code, ok
}
