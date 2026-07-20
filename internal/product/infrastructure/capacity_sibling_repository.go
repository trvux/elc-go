package infrastructure

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/product/domain"
)

const capacityAttributeCode = "phan_khuc_hp"

var capacityDigitsPattern = regexp.MustCompile(`[0-9]+`)

// capacityFamilyKey collapses a variant MPN down to just its non-digit
// characters, upper-cased — e.g. "FTKY25ZVMV" and "FTKY35ZVMV" both become
// "FTKYZVMV". Two products with the same key (within the same brand+
// category) are the same model at a different HP/capacity; this
// deliberately doesn't try to split prefix/digits/suffix into separate
// fields since the merged key is all comparison needs.
func capacityFamilyKey(mpn string) string {
	return capacityDigitsPattern.ReplaceAllString(strings.ToUpper(mpn), "")
}

// attachCapacitySiblings loads other published products that are the same
// model as p at a different capacity — see capacityFamilyKey and
// domain.CapacitySibling's doc comments. Called from GetByID/GetBySlug
// (single-product reads only, same set attachAttributeValuesToProducts
// already covers) — GetByID is what the public detail page actually uses in
// practice, reached via elc-tem's slug_registry → entityId → GetByID
// lookup, not GetBySlug's own route. No-op if p has no variant_mpns to key
// off of. Leaves p.CapacitySiblings nil when the match set is just p itself
// (no real siblings), same "don't render a length-1 selector" convention
// the frontend already applies to the variant option switcher.
func attachCapacitySiblings(ctx context.Context, pool *pgxpool.Pool, p *domain.ProductWithRelations) error {
	mpns := p.VariantMpns()
	if mpns == "" {
		return nil
	}
	familyKey := capacityFamilyKey(mpns)

	rows, err := pool.Query(ctx, `
		SELECT p.id, p.slug, p.name,
		       COALESCE(hp.value_text, p.variant_mpns) AS capacity_label
		FROM products p
		LEFT JOIN LATERAL (
			SELECT pav.value_text
			FROM product_attribute_values pav
			JOIN attribute_definitions ad ON ad.id = pav.attribute_definition_id
				AND ad.code = $4 AND ad.deleted_at IS NULL
			WHERE pav.product_id = p.id AND pav.deleted_at IS NULL
			LIMIT 1
		) hp ON true
		WHERE p.deleted_at IS NULL AND p.status = 'published'
		  AND p.brand_id = $1 AND p.category_id = $2
		  AND regexp_replace(upper(p.variant_mpns), '[0-9]+', '', 'g') = $3
		ORDER BY p.display_price ASC NULLS LAST`,
		p.BrandID(), p.CategoryID(), familyKey, capacityAttributeCode,
	)
	if err != nil {
		return fmt.Errorf("product repository attachCapacitySiblings: %w", err)
	}
	defer rows.Close()

	var siblings []domain.CapacitySibling
	for rows.Next() {
		var s domain.CapacitySibling
		if err := rows.Scan(&s.ID, &s.Slug, &s.Name, &s.CapacityLabel); err != nil {
			return fmt.Errorf("product repository attachCapacitySiblings scan: %w", err)
		}
		s.IsCurrent = s.ID == p.ID()
		siblings = append(siblings, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("product repository attachCapacitySiblings rows: %w", err)
	}

	if len(siblings) > 1 {
		p.CapacitySiblings = siblings
	}
	return nil
}
