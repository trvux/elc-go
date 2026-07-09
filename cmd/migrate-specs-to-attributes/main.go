// One-off migration: best-effort maps each product's old free-text
// products.specs entries onto the new structured product_attribute_values
// (see internal/attribute, scripts/seed-attribute-definitions.sql).
//
// Deliberately conservative — never guesses:
//   - Only top-level spec entries with a plain Value (no nested Items[]) are
//     considered. Items[] was used for two different things in the old data
//     (multi-unit capacity breakdowns AND section headers like "THÔNG TIN
//     DÀN LẠNH") — both are too ambiguous to auto-parse safely, so they're
//     left untouched in the old specs column for manual admin re-entry.
//   - text attributes: copied verbatim (trimmed) once the label matches.
//   - select attributes: only mapped if the trimmed value case-insensitively
//     equals one of the attribute's own options exactly.
//   - number attributes: only mapped if the value parses cleanly as a plain
//     decimal (commas stripped) with nothing else in the string — a
//     compound value like "9,200 (3,400-10,900)" is left alone.
//
// Idempotent: ON CONFLICT on the (product_id, attribute_definition_id)
// partial unique index, safe to re-run.
//
// Usage: DATABASE_URL=... go run ./cmd/migrate-specs-to-attributes
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

// synonymMap: normalized old label -> attribute code, for the AC hardware
// group (5 categories sharing the same code set) — built from the real
// label audit in docs/catalog-audit-2026-07-08.md (100+ label variants
// found for "Máy lạnh treo tường" alone), extended to cover the equivalent
// variants seen across the other 4 AC categories.
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
	"chênh lệch độ cao tối đa":                    "chenh_lech_do_cao_toi_da",
	"chênh lệch độ cao (tối đa)":                  "chenh_lech_do_cao_toi_da",
	"chênh lệch độ cao (tối đa) (m)":              "chenh_lech_do_cao_toi_da",
	"chênh lệch độ cao tối đa giữa dàn nóng-lạnh": "chenh_lech_do_cao_toi_da",
	"chiều cao lắp đặt tối đa giữa dàn nóng-lạnh": "chenh_lech_do_cao_toi_da",
	"độ cao chênh lệch tối đa":                    "chenh_lech_do_cao_toi_da",
	"hiệu suất năng lượng (cspf)":                 "hieu_suat_cspf",
	"hiệu suất năng lượng cspf":                   "hieu_suat_cspf",
	"chỉ số hiệu suất năng lượng (cspf)":          "hieu_suat_cspf",
	"phạm vi làm lạnh hiệu quả":                   "pham_vi_lam_lanh",
	"sử dụng cho phòng":                           "pham_vi_lam_lanh",
	"điện năng tiêu thụ":                          "dien_nang_tieu_thu",
	"điện năng tiêu thụ định mức":                 "dien_nang_tieu_thu",
	"công suất tiêu thụ":                          "dien_nang_tieu_thu",
	"công suất tiêu thụ điện":                     "dien_nang_tieu_thu",
	"nhãn năng lượng tiết kiệm điện":              "nhan_nang_luong",
}

var numberRegex = regexp.MustCompile(`^[0-9][0-9,]*(\.[0-9]+)?$`)

func normalizeLabel(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.ToLower(s)
}

// punctFold strips whitespace/dashes/underscores for a punctuation-insensitive
// compare (e.g. "R-32" vs "R32") — still requires an exact letter/digit
// match, never guesses new characters.
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

// stripOwnUnit removes a trailing occurrence of the attribute's own declared
// unit (e.g. value "50 m" for a field whose unit is "m") before numeric
// parsing. Safe because the unit is already implied by the field itself —
// this removes redundant restated text, it doesn't infer new information.
func stripOwnUnit(value, unit string) string {
	if unit == "" {
		return value
	}
	v := strings.TrimSpace(value)
	// Try "<num> (unit)", "<num>(unit)", "<num> unit", "<num>unit" — longest
	// match first so "(m)" isn't left with a stray paren.
	candidates := []string{
		"(" + unit + ")", unit,
	}
	for _, suf := range candidates {
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

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// Load attribute definitions grouped by category (plus global, category
	// IS NULL, applied to every category).
	rows, err := pool.Query(ctx, `
		SELECT id, category_id, code, name, data_type, unit, options
		FROM attribute_definitions WHERE deleted_at IS NULL`)
	if err != nil {
		log.Fatalf("load definitions: %v", err)
	}
	defsByCategory := map[string][]attributeDef{}
	var globalDefs []attributeDef
	for rows.Next() {
		var id, code, name, dataType string
		var categoryID, unit *string
		var options []string
		if err := rows.Scan(&id, &categoryID, &code, &name, &dataType, &unit, &options); err != nil {
			log.Fatalf("scan definition: %v", err)
		}
		unitStr := ""
		if unit != nil {
			unitStr = *unit
		}
		def := attributeDef{id: id, code: code, name: name, dataType: dataType, unit: unitStr, options: options}
		if categoryID == nil {
			globalDefs = append(globalDefs, def)
		} else {
			defsByCategory[*categoryID] = append(defsByCategory[*categoryID], def)
		}
	}
	rows.Close()

	acCategoryIDs := map[string]bool{}
	acRows, err := pool.Query(ctx, `SELECT id FROM categories WHERE name IN (
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

	// Load products.
	prows, err := pool.Query(ctx, `SELECT id, category_id, specs FROM products WHERE deleted_at IS NULL`)
	if err != nil {
		log.Fatalf("load products: %v", err)
	}
	type productRow struct {
		id, categoryID string
		specs          []byte
	}
	var products []productRow
	for prows.Next() {
		var p productRow
		if err := prows.Scan(&p.id, &p.categoryID, &p.specs); err != nil {
			log.Fatalf("scan product: %v", err)
		}
		products = append(products, p)
	}
	prows.Close()

	totalEntries, matched, skipped := 0, 0, 0

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	for _, p := range products {
		var specs []specItem
		if len(p.specs) == 0 {
			continue
		}
		if err := json.Unmarshal(p.specs, &specs); err != nil {
			log.Printf("product %s: skip, invalid specs json: %v", p.id, err)
			continue
		}

		candidates := append(append([]attributeDef{}, defsByCategory[p.categoryID]...), globalDefs...)
		isAC := acCategoryIDs[p.categoryID]

		for _, item := range specs {
			if item.Value == nil || item.Items != nil {
				continue // ambiguous multi-unit/group-header entry — leave alone
			}
			totalEntries++
			norm := normalizeLabel(item.Label)
			value := strings.TrimSpace(*item.Value)
			if value == "" {
				continue
			}

			var target *attributeDef
			var code string
			if isAC {
				if c, ok := acSynonyms[norm]; ok {
					code = c
				}
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
				skipped++
				continue
			}

			var valueText, valueNumber, valueBoolean any
			switch target.dataType {
			case "text":
				valueText = value
			case "select":
				matchedOption := ""
				normValue := punctFold(value)
				for _, opt := range target.options {
					if punctFold(opt) == normValue {
						matchedOption = opt
						break
					}
				}
				if matchedOption == "" {
					skipped++
					continue
				}
				valueText = matchedOption
			case "number":
				cleaned := strings.ReplaceAll(stripOwnUnit(value, target.unit), ",", "")
				if !numberRegex.MatchString(cleaned) {
					skipped++
					continue
				}
				n, err := strconv.ParseFloat(cleaned, 64)
				if err != nil {
					skipped++
					continue
				}
				valueNumber = n
			case "boolean":
				lower := strings.ToLower(value)
				if lower == "có" || lower == "yes" || lower == "true" {
					valueBoolean = true
				} else if lower == "không" || lower == "no" || lower == "false" {
					valueBoolean = false
				} else {
					skipped++
					continue
				}
			default:
				skipped++
				continue
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO product_attribute_values (product_id, attribute_definition_id, value_text, value_number, value_boolean)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (product_id, attribute_definition_id) WHERE deleted_at IS NULL DO NOTHING`,
				p.id, target.id, valueText, valueNumber, valueBoolean,
			); err != nil {
				log.Fatalf("insert value for product %s attribute %s: %v", p.id, target.code, err)
			}
			matched++
		}
	}

	fmt.Printf("Spec entries examined: %d\nMatched -> product_attribute_values: %d\nLeft unmapped (still in products.specs): %d\n", totalEntries, matched, skipped)

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}
	fmt.Println("Committed.")
}
