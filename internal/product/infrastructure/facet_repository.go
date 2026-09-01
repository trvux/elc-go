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

	var numberCodes, tokenCodes []string
	for _, def := range defs {
		if def.dataType == "number" {
			numberCodes = append(numberCodes, def.code)
		} else {
			tokenCodes = append(tokenCodes, def.code)
		}
	}

	// Batched into at most 2 additional round trips total (one UNION ALL for
	// every number-type definition, one for every select/multiselect/
	// boolean-type definition) instead of one query per definition — a
	// category with N applicable attributes used to mean N sequential round
	// trips here. See docs/rfc/2026-09-02-backend-code-review-round2.md §3.10.
	numberValues, err := r.computeAttributeRangeFacetsBatch(ctx, filter, numberCodes)
	if err != nil {
		return nil, err
	}
	tokenOptions, err := r.computeAttributeTokenFacetsBatch(ctx, filter, tokenCodes)
	if err != nil {
		return nil, err
	}

	facets := make([]domain.AttributeFacet, 0, len(defs))
	for _, def := range defs {
		facet := domain.AttributeFacet{Code: def.code, Name: def.name, GroupLabel: def.groupLabel, DataType: def.dataType, Unit: def.unit}

		if def.dataType == "number" {
			values := numberValues[def.code]
			if len(values) == 0 {
				continue
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
			facet.Min, facet.Max, facet.Buckets = &minV, &maxV, domain.BuildNumberBuckets(values)
			facets = append(facets, facet)
			continue
		}

		options := tokenOptions[def.code]
		if len(options) == 0 {
			continue
		}
		facet.Options = options
		facets = append(facets, facet)
	}
	return facets, nil
}

// computeAttributeTokenFacetsBatch computes the select/multiselect/boolean
// facet options for every code in one round trip: a UNION ALL of one branch
// per code, each branch keeping that attribute's own "exclude own
// dimension" WHERE clause (facetExclude{Attribute: code}) exactly as the
// old per-attribute query did — behavior is unchanged, only the number of
// round trips drops from len(codes) to 1. $N placeholders are numbered
// continuously across every branch via buildFilterConditionsFrom, since
// Postgres placeholders are global to the whole combined statement.
func (r *PostgresProductRepository) computeAttributeTokenFacetsBatch(ctx context.Context, filter domain.ProductFilter, codes []string) (map[string][]domain.AttributeFacetOption, error) {
	result := make(map[string][]domain.AttributeFacetOption, len(codes))
	if len(codes) == 0 {
		return result, nil
	}

	branches := make([]string, 0, len(codes))
	var args []any
	argN := 1
	for _, code := range codes {
		codeArgN := argN
		conditions, condArgs, next := buildFilterConditionsFrom(filter, facetExclude{Attribute: code}, argN+1)
		args = append(args, code)
		args = append(args, condArgs...)
		argN = next

		where := ""
		if len(conditions) > 0 {
			where = " WHERE " + strings.Join(conditions, " AND ")
		}
		branches = append(branches, fmt.Sprintf(
			"SELECT $%d::text AS def_code, unnest(p.facet_tokens) AS token, COUNT(*) AS cnt FROM products p%s GROUP BY token",
			codeArgN, where,
		))
	}

	rows, err := r.pool.Query(ctx, strings.Join(branches, " UNION ALL "), args...)
	if err != nil {
		return nil, fmt.Errorf("product repository computeAttributeTokenFacetsBatch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var code, token string
		var count int
		if err := rows.Scan(&code, &token, &count); err != nil {
			return nil, fmt.Errorf("product repository computeAttributeTokenFacetsBatch scan: %w", err)
		}
		if value, ok := strings.CutPrefix(token, code+":"); ok {
			result[code] = append(result[code], domain.AttributeFacetOption{Value: value, Count: count})
		}
	}
	return result, rows.Err()
}

// computeAttributeRangeFacetsBatch fetches every matching value for every
// number-type code in one round trip (UNION ALL, same "exclude own
// dimension" semantics per branch as the old per-attribute query — see
// computeAttributeTokenFacetsBatch's doc comment for the same reasoning).
// Returns the raw matching values per code; the caller computes min/max and
// calls domain.BuildNumberBuckets, same as before.
func (r *PostgresProductRepository) computeAttributeRangeFacetsBatch(ctx context.Context, filter domain.ProductFilter, codes []string) (map[string][]float64, error) {
	result := make(map[string][]float64, len(codes))
	if len(codes) == 0 {
		return result, nil
	}

	branches := make([]string, 0, len(codes))
	var args []any
	argN := 1
	for _, code := range codes {
		codeArgN := argN
		conditions, condArgs, next := buildFilterConditionsFrom(filter, facetExclude{Attribute: code}, argN+1)
		args = append(args, code)
		args = append(args, condArgs...)
		argN = next

		where := strings.Join(append([]string{
			"pav.deleted_at IS NULL", "pav.value_number IS NOT NULL", fmt.Sprintf("ad2.code = $%d", codeArgN),
		}, conditions...), " AND ")
		branches = append(branches, fmt.Sprintf(`SELECT ad2.code AS def_code, pav.value_number AS val
			FROM product_attribute_values pav
			JOIN attribute_definitions ad2 ON ad2.id = pav.attribute_definition_id
			JOIN products p ON p.id = pav.product_id
			WHERE %s`, where))
	}

	rows, err := r.pool.Query(ctx, strings.Join(branches, " UNION ALL "), args...)
	if err != nil {
		return nil, fmt.Errorf("product repository computeAttributeRangeFacetsBatch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var code string
		var val float64
		if err := rows.Scan(&code, &val); err != nil {
			return nil, fmt.Errorf("product repository computeAttributeRangeFacetsBatch scan: %w", err)
		}
		result[code] = append(result[code], val)
	}
	return result, rows.Err()
}
