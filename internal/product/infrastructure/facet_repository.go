package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/trvux/elc-go/internal/product/domain"
)

// RecomputeFacetTokens rebuilds a product's denormalized facet_tokens cache
// from the current state of product_attribute_values — called after every
// write to a product's attribute values, inside the same transaction. Same
// "compute once at write time, read via a plain column" technique already
// used by RecomputeDisplayCache (variant_repository.go) for
// display_price/price_min/price_max/variant_mpns.
//
// Only select/multiselect/boolean values become tokens ("code:value") —
// number-type attributes (BTU, HP, ...) are range-filterable, not
// discrete/equality-facetable, so filtering/faceting for those goes directly
// against product_attribute_values.value_number instead of this array.
func RecomputeFacetTokens(ctx context.Context, tx pgx.Tx, productID string) error {
	_, err := tx.Exec(ctx, `
		WITH tokens AS (
			SELECT ad.code || ':' || opt AS token
			FROM product_attribute_values pav
			JOIN attribute_definitions ad ON ad.id = pav.attribute_definition_id
			CROSS JOIN LATERAL unnest(
				CASE ad.data_type
					WHEN 'select' THEN ARRAY[pav.value_text]
					WHEN 'multiselect' THEN pav.value_options
					WHEN 'boolean' THEN ARRAY[pav.value_boolean::text]
					ELSE ARRAY[]::text[]
				END
			) AS opt
			WHERE pav.product_id = $1 AND pav.deleted_at IS NULL AND opt IS NOT NULL
		)
		UPDATE products
		SET facet_tokens = COALESCE((SELECT array_agg(token) FROM tokens), '{}')
		WHERE id = $1`,
		productID,
	)
	if err != nil {
		return fmt.Errorf("product repository RecomputeFacetTokens: %w", err)
	}
	return nil
}

// computeBrandFacets counts published-under-filter products per brand,
// excluding the filter's own brand dimension.
func (r *PostgresProductRepository) computeBrandFacets(ctx context.Context, filter domain.ProductFilter) ([]domain.BrandFacet, error) {
	conditions, args := buildFilterConditions(filter, facetExclude{Brand: true})
	query := `SELECT br.id, br.name, br.slug, br.logo_url, COUNT(*)
		FROM products p
		LEFT JOIN brands br ON br.id = p.brand_id`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " GROUP BY br.id, br.name, br.slug, br.logo_url ORDER BY br.name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("product repository computeBrandFacets: %w", err)
	}
	defer rows.Close()

	var facets []domain.BrandFacet
	for rows.Next() {
		var f domain.BrandFacet
		if err := rows.Scan(&f.ID, &f.Name, &f.Slug, &f.LogoURL, &f.Count); err != nil {
			return nil, fmt.Errorf("product repository computeBrandFacets scan: %w", err)
		}
		facets = append(facets, f)
	}
	return facets, rows.Err()
}

// computePriceFacets returns the min/max display_price under the filter
// (excluding the filter's own price dimension) plus ready-to-click bucket
// suggestions built from every matching product's actual price — see
// domain.BuildNumberBuckets.
func (r *PostgresProductRepository) computePriceFacets(ctx context.Context, filter domain.ProductFilter) (domain.PriceFacet, error) {
	conditions, args := buildFilterConditions(filter, facetExclude{Price: true})
	// display_price = 0/NULL means "Liên hệ" (quote-only, no public price) —
	// excluded from bucketing, same sentinel-handling rule already used for
	// JSON-LD Offer generation (see elc_new_product_model_decided memory).
	query := `SELECT p.display_price FROM products p WHERE p.display_price IS NOT NULL AND p.display_price > 0`
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return domain.PriceFacet{}, fmt.Errorf("product repository computePriceFacets: %w", err)
	}
	defer rows.Close()

	var values []float64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return domain.PriceFacet{}, fmt.Errorf("product repository computePriceFacets scan: %w", err)
		}
		values = append(values, float64(v))
	}
	if err := rows.Err(); err != nil {
		return domain.PriceFacet{}, fmt.Errorf("product repository computePriceFacets rows: %w", err)
	}
	if len(values) == 0 {
		return domain.PriceFacet{}, nil
	}

	buckets := domain.BuildPriceBuckets(values)
	f := domain.PriceFacet{Buckets: buckets}
	for _, v := range values {
		iv := int64(v)
		if f.Min == 0 || iv < f.Min {
			f.Min = iv
		}
		if iv > f.Max {
			f.Max = iv
		}
	}
	return f, nil
}

// computeAttributeFacets builds one AttributeFacet per attribute definition
// applicable to the categories currently represented on the page (the
// *page-level scope* — category/brand/product-line/featured/status only,
// deliberately not narrowed by search/price/other-attribute filters, so the
// set of filter *controls* shown stays stable while the user toggles
// individual options; only each control's own counts shrink). For each
// applicable definition: select/multiselect/boolean use the denormalized
// facet_tokens array (grouped-count, exclude-own-dimension); number uses a
// direct MIN/MAX against product_attribute_values (range, not
// token-facetable).
func (r *PostgresProductRepository) computeAttributeFacets(ctx context.Context, filter domain.ProductFilter) ([]domain.AttributeFacet, error) {
	scopeFilter := domain.ProductFilter{
		CategoryID: filter.CategoryID, CategoryIDs: filter.CategoryIDs,
		BrandID: filter.BrandID, BrandIDs: filter.BrandIDs,
		ProductLineID: filter.ProductLineID, IsFeatured: filter.IsFeatured,
		Status: filter.Status, IncludeDeleted: filter.IncludeDeleted,
	}
	conditions, args := buildFilterConditions(scopeFilter, facetExclude{})
	categoryQuery := "SELECT DISTINCT p.category_id FROM products p"
	if len(conditions) > 0 {
		categoryQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	catRows, err := r.pool.Query(ctx, categoryQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("product repository computeAttributeFacets (categories): %w", err)
	}
	var categoryIDs []string
	for catRows.Next() {
		var id string
		if err := catRows.Scan(&id); err != nil {
			catRows.Close()
			return nil, fmt.Errorf("product repository computeAttributeFacets (categories) scan: %w", err)
		}
		categoryIDs = append(categoryIDs, id)
	}
	catRows.Close()
	if err := catRows.Err(); err != nil {
		return nil, fmt.Errorf("product repository computeAttributeFacets (categories) rows: %w", err)
	}
	if len(categoryIDs) == 0 {
		return nil, nil
	}

	// Applicable definitions: attached to any represented category, or
	// global (attached to zero categories) — plain SQL join into
	// attribute_definitions/category_attribute_definitions, same
	// cross-module-via-SQL precedent as attachAttributeValuesToProducts.
	defRows, err := r.pool.Query(ctx, `
		SELECT ad.id, ad.code, ad.name, ad.group_label, ad.data_type, ad.unit
		FROM attribute_definitions ad
		WHERE ad.deleted_at IS NULL
		  AND (
		    EXISTS (SELECT 1 FROM category_attribute_definitions cad WHERE cad.attribute_definition_id = ad.id AND cad.category_id = ANY($1::uuid[]))
		    OR NOT EXISTS (SELECT 1 FROM category_attribute_definitions cad WHERE cad.attribute_definition_id = ad.id)
		  )
		ORDER BY ad.order_index ASC`, categoryIDs)
	if err != nil {
		return nil, fmt.Errorf("product repository computeAttributeFacets (definitions): %w", err)
	}
	type definitionRow struct {
		id, code, name, dataType string
		groupLabel, unit         *string
	}
	var defs []definitionRow
	for defRows.Next() {
		var d definitionRow
		if err := defRows.Scan(&d.id, &d.code, &d.name, &d.groupLabel, &d.dataType, &d.unit); err != nil {
			defRows.Close()
			return nil, fmt.Errorf("product repository computeAttributeFacets (definitions) scan: %w", err)
		}
		defs = append(defs, d)
	}
	defRows.Close()
	if err := defRows.Err(); err != nil {
		return nil, fmt.Errorf("product repository computeAttributeFacets (definitions) rows: %w", err)
	}

	facets := make([]domain.AttributeFacet, 0, len(defs))
	for _, def := range defs {
		facet := domain.AttributeFacet{Code: def.code, Name: def.name, GroupLabel: def.groupLabel, DataType: def.dataType, Unit: def.unit}

		if def.dataType == "number" {
			min, max, buckets, err := r.computeAttributeRangeFacet(ctx, filter, def.code)
			if err != nil {
				return nil, err
			}
			if min == nil {
				continue
			}
			facet.Min, facet.Max, facet.Buckets = min, max, buckets
			facets = append(facets, facet)
			continue
		}

		options, err := r.computeAttributeTokenFacet(ctx, filter, def.code)
		if err != nil {
			return nil, err
		}
		if len(options) == 0 {
			continue
		}
		facet.Options = options
		facets = append(facets, facet)
	}
	return facets, nil
}

func (r *PostgresProductRepository) computeAttributeTokenFacet(ctx context.Context, filter domain.ProductFilter, code string) ([]domain.AttributeFacetOption, error) {
	conditions, args := buildFilterConditions(filter, facetExclude{Attribute: code})
	query := "SELECT unnest(p.facet_tokens) AS token, COUNT(*) FROM products p"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " GROUP BY token"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("product repository computeAttributeTokenFacet: %w", err)
	}
	defer rows.Close()

	prefix := code + ":"
	var options []domain.AttributeFacetOption
	for rows.Next() {
		var token string
		var count int
		if err := rows.Scan(&token, &count); err != nil {
			return nil, fmt.Errorf("product repository computeAttributeTokenFacet scan: %w", err)
		}
		if value, ok := strings.CutPrefix(token, prefix); ok {
			options = append(options, domain.AttributeFacetOption{Value: value, Count: count})
		}
	}
	return options, rows.Err()
}

// computeAttributeRangeFacet fetches every matching value for a number-type
// attribute (excluding the attribute's own filter dimension) and returns
// both the overall min/max and ready-to-click bucket suggestions built from
// the real data — see domain.BuildNumberBuckets.
func (r *PostgresProductRepository) computeAttributeRangeFacet(ctx context.Context, filter domain.ProductFilter, code string) (min, max *float64, buckets []domain.NumberBucket, err error) {
	conditions, args := buildFilterConditions(filter, facetExclude{Attribute: code})
	argN := len(args) + 1
	query := fmt.Sprintf(`
		SELECT pav.value_number
		FROM product_attribute_values pav
		JOIN attribute_definitions ad2 ON ad2.id = pav.attribute_definition_id
		JOIN products p ON p.id = pav.product_id
		WHERE pav.deleted_at IS NULL AND pav.value_number IS NOT NULL AND ad2.code = $%d`, argN)
	args = append(args, code)
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	rows, queryErr := r.pool.Query(ctx, query, args...)
	if queryErr != nil {
		return nil, nil, nil, fmt.Errorf("product repository computeAttributeRangeFacet: %w", queryErr)
	}
	defer rows.Close()

	var values []float64
	for rows.Next() {
		var v float64
		if scanErr := rows.Scan(&v); scanErr != nil {
			return nil, nil, nil, fmt.Errorf("product repository computeAttributeRangeFacet scan: %w", scanErr)
		}
		values = append(values, v)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, nil, nil, fmt.Errorf("product repository computeAttributeRangeFacet rows: %w", rowsErr)
	}
	if len(values) == 0 {
		return nil, nil, nil, nil
	}

	minV, maxV := values[0], values[0]
	for _, v := range values {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	return &minV, &maxV, domain.BuildNumberBuckets(values), nil
}
