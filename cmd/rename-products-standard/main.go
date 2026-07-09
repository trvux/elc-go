// One-off: renames products to match the naming convention confirmed via
// web research 2026-07-09 against real competitors (dienmayxanh.com,
// dienmaycholon.com, hacome.vn, mecosun.vn, nexttechvn.com, etc.):
//
//   - Daikin/LG wall-mount (treo tường): "Máy lạnh {Brand} Inverter
//     [{1/2} chiều] {HP} {MPN}" — installation type omitted (implied
//     default, matches every competitor title found).
//   - Daikin/LG other AC categories (ducted/ceiling/cassette/floor-
//     standing): "Máy lạnh {install type} {Brand} {MPN} {HP} - Loại
//     {chiều}, {Inverter/Không Inverter}" — competitors always name the
//     install type here since it's not the default.
//   - Menred / Acis: brand name appended as "(Brand)" if not already
//     present in the name — real Menred resellers (mecosun.vn) lead with
//     a descriptive phrase then the brand+model, and Acis resellers
//     (nexttechvn.com) already match our existing descriptive style
//     closely, just missing the brand name in the title text itself.
//
// HP/chiều are extracted from the CURRENT name text (regex), not
// attribute_values — most products don't have cong_suat_lam_lanh_btu/hp
// populated yet (only 790/2042 legacy specs entries have been migrated),
// so re-deriving from mostly-empty structured data would be less
// reliable than the free-text extraction the frontend's own
// extractProductHp() already relies on.
//
// New slug = slugify(new name) directly — replaces the old
// slug(name)+"-"+slug(mpn) concatenation (see useProductForm.ts
// regenerateSlug), since mpn is now a first-class part of the name
// itself, not bolted on separately.
//
// Prints old_slug -> new_slug for every changed product (for the
// next.config.ts redirect list) and writes name/slug in one transaction.
//
// Usage: DATABASE_URL=... go run ./cmd/rename-products-standard [-commit]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type product struct {
	id, name, slug, categoryName, brandName, mpn, productLineCode string
}

var hpRegex = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*HP`)
var chieuRegex = regexp.MustCompile(`(?i)(một|hai)\s*chiều`)

// install type phrase per category — matches real competitor titles
// (e.g. "Điều hòa âm trần cassette Daikin FCF140CVM...").
var installTypeByCategory = map[string]string{
	"Máy lạnh âm trần đa hướng thổi":  "âm trần cassette",
	"Máy lạnh áp trần":                "áp trần",
	"Máy lạnh giấu trần nối ống gió":  "giấu trần nối ống gió",
	"Máy lạnh tủ đứng":                "tủ đứng",
}

// product_line codes seeded as the non-inverter tier for their category
// (backfill-daikin-remaining-categories.sql) — everything else in an AC
// category is inverter.
var nonInverterLineCodes = map[string]bool{"FDNQ": true, "FHNQ": true, "FCNQ": true}

func slugify(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		out = s
	}
	out = strings.ToLower(out)
	out = strings.ReplaceAll(out, "đ", "d")
	out = regexp.MustCompile(`[^a-z0-9\s-]`).ReplaceAllString(out, "")
	out = regexp.MustCompile(`\s+`).ReplaceAllString(out, "-")
	out = regexp.MustCompile(`-+`).ReplaceAllString(out, "-")
	return strings.Trim(out, "-")
}

func buildACName(p product) string {
	hp := ""
	if m := hpRegex.FindStringSubmatch(p.name); m != nil {
		hp = m[1] + "HP"
	}
	chieu := ""
	if m := chieuRegex.FindStringSubmatch(p.name); m != nil {
		chieu = strings.ToLower(m[1]) + " chiều"
	}
	invert := "Inverter"
	if nonInverterLineCodes[p.productLineCode] {
		invert = "Không Inverter"
	}

	if p.categoryName == "Máy lạnh treo tường" {
		parts := []string{"Máy lạnh", p.brandName, invert}
		if chieu != "" {
			parts = append(parts, chieu)
		}
		if hp != "" {
			parts = append(parts, hp)
		}
		parts = append(parts, p.mpn)
		return strings.Join(parts, " ")
	}

	installType := installTypeByCategory[p.categoryName]
	parts := []string{"Máy lạnh", installType, p.brandName, p.mpn}
	if hp != "" {
		parts = append(parts, hp)
	}
	tail := "- Loại"
	if chieu != "" {
		tail += " " + chieu + ","
	}
	tail += " " + invert
	return strings.Join(parts, " ") + " " + tail
}

func buildBrandAppendedName(p product) string {
	if strings.Contains(strings.ToLower(p.name), strings.ToLower(p.brandName)) {
		return p.name
	}
	return fmt.Sprintf("%s (%s)", strings.TrimSpace(p.name), p.brandName)
}

func main() {
	commit := flag.Bool("commit", false, "actually write changes (default: dry run, print only)")
	flag.Parse()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `
		SELECT p.id, p.name, p.slug, c.name, b.name, v.mpn, COALESCE(pl.code, '')
		FROM products p
		JOIN categories c ON c.id = p.category_id
		JOIN brands b ON b.id = p.brand_id
		JOIN product_variants v ON v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		WHERE p.deleted_at IS NULL
		ORDER BY b.name, c.name, v.mpn`)
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	var products []product
	for rows.Next() {
		var p product
		if err := rows.Scan(&p.id, &p.name, &p.slug, &p.categoryName, &p.brandName, &p.mpn, &p.productLineCode); err != nil {
			log.Fatalf("scan: %v", err)
		}
		products = append(products, p)
	}
	rows.Close()

	acCategories := map[string]bool{
		"Máy lạnh treo tường": true, "Máy lạnh âm trần đa hướng thổi": true,
		"Máy lạnh áp trần": true, "Máy lạnh giấu trần nối ống gió": true, "Máy lạnh tủ đứng": true,
	}

	type change struct {
		id, oldSlug, newSlug, oldName, newName string
	}
	var changes []change
	seenSlug := map[string]string{} // dedupe guard: new slug -> product id

	for _, p := range products {
		var newName string
		if acCategories[p.categoryName] && (p.brandName == "Daikin" || p.brandName == "LG") {
			newName = buildACName(p)
		} else {
			newName = buildBrandAppendedName(p)
		}
		if newName == p.name {
			continue
		}
		newSlug := slugify(newName)
		if owner, exists := seenSlug[newSlug]; exists {
			fmt.Printf("SKIP (slug collision with %s): %s -> %q\n", owner, p.id, newName)
			continue
		}
		seenSlug[newSlug] = p.id
		changes = append(changes, change{p.id, p.slug, newSlug, p.name, newName})
	}

	fmt.Printf("=== %d products would be renamed (of %d total) ===\n\n", len(changes), len(products))
	for _, c := range changes {
		fmt.Printf("%s\n  name: %q\n       -> %q\n  slug: %s -> %s\n\n", c.id, c.oldName, c.newName, c.oldSlug, c.newSlug)
	}

	if !*commit {
		fmt.Println("Dry run only (pass -commit to write). No changes made.")
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)
	for _, c := range changes {
		if _, err := tx.Exec(ctx, `UPDATE products SET name = $1, slug = $2 WHERE id = $3`, c.newName, c.newSlug, c.id); err != nil {
			log.Fatalf("update product %s: %v", c.id, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}
	fmt.Printf("Committed %d renames.\n", len(changes))

	fmt.Println("\n=== redirect pairs (old_slug -> new_slug) for next.config.ts ===")
	for _, c := range changes {
		fmt.Printf("%s -> %s\n", c.oldSlug, c.newSlug)
	}
}
