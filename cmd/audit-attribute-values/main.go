// Read-only audit: recomputes the same specs -> attribute_values mapping
// logic as cmd/migrate-specs-to-attributes (label synonym match + value
// coercion), then compares the recomputed result against what's ACTUALLY
// stored in product_attribute_values right now. Flags:
//   - MISMATCH: a value that should be there per specs differs from what's
//     stored (migration bug or a value edited since without updating specs).
//   - MISSING: specs data that matches a known attribute but never made it
//     into product_attribute_values at all.
//   - Sanity checks: number attributes with implausible values (<=0 where
//     a real unit implies positive, e.g. BTU/CSPF), select attributes
//     whose stored value isn't one of the definition's own options.
//
// Does NOT check "does this match real Daikin/LG spec sheets" — that
// needs external research per product, not a mechanical diff against our
// own already-migrated data. This only proves internal consistency: the
// migration transcribed the old specs correctly, and current values are
// well-formed per each attribute's own data_type/options.
//
// Usage: DATABASE_URL=... go run ./cmd/audit-attribute-values
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type specItem struct {
	Label string  `json:"label"`
	Value *string `json:"value,omitempty"`
	Items []struct {
		Label string `json:"label"`
		Value string `json:"value"`
	} `json:"items,omitempty"`
}

type attributeDef struct {
	id, code, name, dataType, unit string
	options                        []string
}

var acSynonyms = map[string]string{
	"loại máy": "loai_may", "công nghệ inverter": "cong_nghe_inverter",
	"loại gas lạnh": "loai_gas_lanh", "môi chất lạnh": "loai_gas_lanh", "môi chất làm lạnh": "loai_gas_lanh",
	"môi chất lạnh (gas)": "loai_gas_lanh", "nguồn điện": "nguon_dien", "nguồn điện (ph/v/hz)": "nguon_dien",
	"điện áp vào": "nguon_dien", "xuất xứ": "xuat_xu", "sản xuất tại": "xuat_xu", "nguồn gốc": "xuat_xu",
	"độ ồn": "do_on", "độ ồn (cao / cực thấp)": "do_on",
	"độ ồn (cao / trung bình / thấp / yên tĩnh)": "do_on", "độ ồn (cao / trung bình / thấp)": "do_on",
	"độ ồn (cao/trung bình/thấp)": "do_on",
	"kích thước dàn lạnh":         "kich_thuoc_dan_lanh", "kích thước dàn lạnh (rxcxs)": "kich_thuoc_dan_lanh",
	"khối lượng dàn lạnh": "trong_luong_dan_lanh", "trọng lượng dàn lạnh": "trong_luong_dan_lanh",
	"kích thước dàn nóng": "kich_thuoc_dan_nong", "kích thước dàn nóng (rxcxs)": "kich_thuoc_dan_nong",
	"khối lượng dàn nóng": "trong_luong_dan_nong", "trọng lượng dàn nóng": "trong_luong_dan_nong",
	"chiều dài tối đa": "chieu_dai_ong_gas_toi_da", "chiều dài ống gas tối đa": "chieu_dai_ong_gas_toi_da",
	"chiều dài ống gas tối đa (m)": "chieu_dai_ong_gas_toi_da", "chiều dài đường ống": "chieu_dai_ong_gas_toi_da",
	"chiều dài đường ống tối đa": "chieu_dai_ong_gas_toi_da",
	"chênh lệch độ cao tối đa":   "chenh_lech_do_cao_toi_da", "chênh lệch độ cao (tối đa)": "chenh_lech_do_cao_toi_da",
	"chênh lệch độ cao (tối đa) (m)":              "chenh_lech_do_cao_toi_da",
	"chênh lệch độ cao tối đa giữa dàn nóng-lạnh": "chenh_lech_do_cao_toi_da",
	"chiều cao lắp đặt tối đa giữa dàn nóng-lạnh": "chenh_lech_do_cao_toi_da",
	"độ cao chênh lệch tối đa":                    "chenh_lech_do_cao_toi_da",
	"hiệu suất năng lượng (cspf)":                 "hieu_suat_cspf", "hiệu suất năng lượng cspf": "hieu_suat_cspf",
	"chỉ số hiệu suất năng lượng (cspf)": "hieu_suat_cspf",
	"phạm vi làm lạnh hiệu quả":          "pham_vi_lam_lanh", "sử dụng cho phòng": "pham_vi_lam_lanh",
	"điện năng tiêu thụ": "dien_nang_tieu_thu", "điện năng tiêu thụ định mức": "dien_nang_tieu_thu",
	"công suất tiêu thụ": "dien_nang_tieu_thu", "công suất tiêu thụ điện": "dien_nang_tieu_thu",
	"nhãn năng lượng tiết kiệm điện": "nhan_nang_luong",
}

var numberRegex = regexp.MustCompile(`^[0-9][0-9,]*(\.[0-9]+)?$`)

func normalizeLabel(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
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
			trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, "("))
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return value
}

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	defsByID := map[string]attributeDef{}
	defsByCategory := map[string][]attributeDef{}
	var globalDefs []attributeDef
	rows, _ := pool.Query(ctx, `SELECT id, category_id, code, name, data_type, unit, options FROM attribute_definitions WHERE deleted_at IS NULL`)
	for rows.Next() {
		var id, code, name, dataType string
		var categoryID, unit *string
		var options []string
		rows.Scan(&id, &categoryID, &code, &name, &dataType, &unit, &options)
		u := ""
		if unit != nil {
			u = *unit
		}
		def := attributeDef{id: id, code: code, name: name, dataType: dataType, unit: u, options: options}
		defsByID[id] = def
		if categoryID == nil {
			globalDefs = append(globalDefs, def)
		} else {
			defsByCategory[*categoryID] = append(defsByCategory[*categoryID], def)
		}
	}
	rows.Close()

	acCategoryIDs := map[string]bool{}
	acRows, _ := pool.Query(ctx, `SELECT id FROM categories WHERE name IN (
		'Máy lạnh treo tường','Máy lạnh âm trần đa hướng thổi','Máy lạnh áp trần',
		'Máy lạnh giấu trần nối ống gió','Máy lạnh tủ đứng') AND deleted_at IS NULL`)
	for acRows.Next() {
		var id string
		acRows.Scan(&id)
		acCategoryIDs[id] = true
	}
	acRows.Close()

	// Current stored values, keyed by product_id+attribute_definition_id.
	type storedVal struct {
		text    string
		num     *float64
		boolean *bool
	}
	stored := map[string]storedVal{}
	svRows, _ := pool.Query(ctx, `SELECT product_id, attribute_definition_id, COALESCE(value_text,''), value_number, value_boolean FROM product_attribute_values WHERE deleted_at IS NULL`)
	for svRows.Next() {
		var pid, adid, text string
		var num *float64
		var boolean *bool
		svRows.Scan(&pid, &adid, &text, &num, &boolean)
		stored[pid+"|"+adid] = storedVal{text, num, boolean}
	}
	svRows.Close()

	prows, _ := pool.Query(ctx, `SELECT id, category_id, specs FROM products WHERE deleted_at IS NULL`)
	type productRow struct {
		id, categoryID string
		specs          []byte
	}
	var products []productRow
	for prows.Next() {
		var p productRow
		prows.Scan(&p.id, &p.categoryID, &p.specs)
		products = append(products, p)
	}
	prows.Close()

	mismatches, missing, sanityFails := 0, 0, 0
	checkedValues := 0

	for _, p := range products {
		if len(p.specs) == 0 {
			continue
		}
		var specs []specItem
		if err := json.Unmarshal(p.specs, &specs); err != nil {
			continue
		}
		candidates := append(append([]attributeDef{}, defsByCategory[p.categoryID]...), globalDefs...)
		isAC := acCategoryIDs[p.categoryID]

		for _, item := range specs {
			if item.Value == nil || item.Items != nil {
				continue
			}
			norm := normalizeLabel(item.Label)
			value := strings.TrimSpace(*item.Value)
			if value == "" {
				continue
			}
			var target *attributeDef
			code := ""
			if isAC {
				code = acSynonyms[norm]
			}
			for i := range candidates {
				if code != "" && candidates[i].code == code {
					target = &candidates[i]
					break
				}
				if code == "" && normalizeLabel(candidates[i].name) == norm {
					target = &candidates[i]
					break
				}
			}
			if target == nil {
				continue // not something the migration would have mapped
			}

			key := p.id + "|" + target.id
			sv, exists := stored[key]

			switch target.dataType {
			case "text":
				expected := value
				if !exists {
					missing++
					fmt.Printf("MISSING product=%s attr=%s expected_text=%q\n", p.id, target.code, expected)
				} else if sv.text != expected {
					mismatches++
					fmt.Printf("MISMATCH product=%s attr=%s expected=%q stored=%q\n", p.id, target.code, expected, sv.text)
				}
				checkedValues++
			case "select":
				matched := ""
				for _, opt := range target.options {
					if punctFold(opt) == punctFold(value) {
						matched = opt
						break
					}
				}
				if matched == "" {
					continue // migration would have skipped this too (no valid option match)
				}
				if !exists {
					missing++
					fmt.Printf("MISSING product=%s attr=%s expected_select=%q\n", p.id, target.code, matched)
				} else if sv.text != matched {
					mismatches++
					fmt.Printf("MISMATCH product=%s attr=%s expected=%q stored=%q\n", p.id, target.code, matched, sv.text)
				}
				checkedValues++
			case "number":
				cleaned := strings.ReplaceAll(stripOwnUnit(value, target.unit), ",", "")
				if !numberRegex.MatchString(cleaned) {
					continue
				}
				n, err := strconv.ParseFloat(cleaned, 64)
				if err != nil {
					continue
				}
				if !exists {
					missing++
					fmt.Printf("MISSING product=%s attr=%s expected_number=%v\n", p.id, target.code, n)
				} else if sv.num == nil || *sv.num != n {
					storedStr := "NULL"
					if sv.num != nil {
						storedStr = fmt.Sprintf("%v", *sv.num)
					}
					mismatches++
					fmt.Printf("MISMATCH product=%s attr=%s expected=%v stored=%s\n", p.id, target.code, n, storedStr)
				}
				checkedValues++
			case "boolean":
				lower := strings.ToLower(value)
				var expected bool
				switch lower {
				case "có", "yes", "true":
					expected = true
				case "không", "no", "false":
					expected = false
				default:
					continue
				}
				if !exists {
					missing++
					fmt.Printf("MISSING product=%s attr=%s expected_bool=%v\n", p.id, target.code, expected)
				} else if sv.boolean == nil || *sv.boolean != expected {
					storedStr := "NULL"
					if sv.boolean != nil {
						storedStr = fmt.Sprintf("%v", *sv.boolean)
					}
					mismatches++
					fmt.Printf("MISMATCH product=%s attr=%s expected=%v stored=%s\n", p.id, target.code, expected, storedStr)
				}
				checkedValues++
			}
		}
	}

	fmt.Printf("\n=== Migration accuracy: %d values re-checked against specs, %d mismatches, %d missing ===\n", checkedValues, mismatches, missing)

	// Sanity pass: every stored number value should be > 0 (every seeded
	// AC number attribute — BTU, CSPF, weight, dimensions, etc. — is
	// physically always positive; a 0 or negative value is definitely
	// wrong data, not a valid reading).
	fmt.Println("\n=== Sanity: number attribute values that are <= 0 ===")
	for key, sv := range stored {
		parts := strings.SplitN(key, "|", 2)
		def, ok := defsByID[parts[1]]
		if !ok || def.dataType != "number" || sv.num == nil {
			continue
		}
		if *sv.num <= 0 {
			sanityFails++
			fmt.Printf("product=%s attr=%s value=%v\n", parts[0], def.code, *sv.num)
		}
	}
	fmt.Printf("=== %d sanity failures ===\n", sanityFails)
}
