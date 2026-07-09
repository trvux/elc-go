package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product/domain"
)

// fetchOptionsForProduct/fetchVariantsForProduct load a single product's full
// variant tree — used ONLY by single-product reads (GetByID/GetBySlug), NEVER
// by list/grid reads (GetAll/queryProducts), which must stay a join-free
// single-table scan on products — see docs/product-v2-design.md's
// performance section and productJoin's doc comment. Separate queries per
// child collection (not one big JOIN) for the same row-multiplication reason
// fetchTagsForProducts is separate from the category/brand JOIN.
func fetchOptionsForProduct(ctx context.Context, pool *pgxpool.Pool, productID string) ([]domain.ProductOption, error) {
	rows, err := pool.Query(ctx, `
		SELECT po.id, po.name, po.order_index, pov.id, pov.value, pov.order_index
		FROM product_options po
		LEFT JOIN product_option_values pov ON pov.option_id = po.id
		WHERE po.product_id = $1
		ORDER BY po.order_index, pov.order_index`, productID)
	if err != nil {
		return nil, fmt.Errorf("product repository fetchOptions: %w", err)
	}
	defer rows.Close()

	var order []string
	byID := map[string]*domain.ProductOption{}
	for rows.Next() {
		var optID, optName string
		var optOrder int
		var valID, valValue *string
		var valOrder *int
		if err := rows.Scan(&optID, &optName, &optOrder, &valID, &valValue, &valOrder); err != nil {
			return nil, fmt.Errorf("product repository fetchOptions scan: %w", err)
		}
		opt, ok := byID[optID]
		if !ok {
			opt = &domain.ProductOption{ID: optID, ProductID: productID, Name: optName, OrderIndex: optOrder}
			byID[optID] = opt
			order = append(order, optID)
		}
		if valID != nil {
			opt.Values = append(opt.Values, domain.ProductOptionValue{
				ID: *valID, Value: derefStr(valValue), OrderIndex: derefInt(valOrder),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("product repository fetchOptions rows: %w", err)
	}

	result := make([]domain.ProductOption, 0, len(order))
	for _, id := range order {
		result = append(result, *byID[id])
	}
	return result, nil
}

func fetchVariantsForProduct(ctx context.Context, pool *pgxpool.Pool, productID string) ([]*domain.ProductVariant, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, mpn, sku, gtin, is_default, is_standalone, stock_status::text, lead_time_days, cost_price,
		       original_price, sale_price, discount_percent, weight, is_active, order_index,
		       created_at, updated_at, deleted_at
		FROM product_variants
		WHERE product_id = $1 AND deleted_at IS NULL
		ORDER BY order_index`, productID)
	if err != nil {
		return nil, fmt.Errorf("product repository fetchVariants: %w", err)
	}
	defer rows.Close()

	var scanned []scannedVariant
	var variantIDs []string
	for rows.Next() {
		var v scannedVariant
		if err := rows.Scan(
			&v.id, &v.mpn, &v.sku, &v.gtin, &v.isDefault, &v.isStandalone, &v.stockStatus, &v.leadTimeDays, &v.costPrice,
			&v.originalPrice, &v.salePrice, &v.discountPercent, &v.weight, &v.isActive, &v.orderIndex,
			&v.createdAt, &v.updatedAt, &v.deletedAt,
		); err != nil {
			return nil, fmt.Errorf("product repository fetchVariants scan: %w", err)
		}
		scanned = append(scanned, v)
		variantIDs = append(variantIDs, v.id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("product repository fetchVariants rows: %w", err)
	}
	if len(scanned) == 0 {
		return nil, nil
	}

	optionValueIDs, err := fetchVariantOptionValueIDs(ctx, pool, variantIDs)
	if err != nil {
		return nil, err
	}
	components, err := fetchVariantComponents(ctx, pool, variantIDs)
	if err != nil {
		return nil, err
	}

	variants := make([]*domain.ProductVariant, len(scanned))
	for i, v := range scanned {
		variants[i] = domain.RehydrateProductVariant(
			v.id, productID, v.mpn, v.sku, v.gtin,
			v.isDefault, v.isStandalone, v.stockStatus, v.leadTimeDays, v.costPrice,
			v.originalPrice, v.salePrice, v.discountPercent, v.weight, v.isActive, v.orderIndex,
			optionValueIDs[v.id], components[v.id],
			v.createdAt, v.updatedAt, v.deletedAt,
		)
	}
	return variants, nil
}

func fetchVariantOptionValueIDs(ctx context.Context, pool *pgxpool.Pool, variantIDs []string) (map[string][]string, error) {
	rows, err := pool.Query(ctx, "SELECT variant_id, option_value_id FROM product_variant_option_values WHERE variant_id = ANY($1)", variantIDs)
	if err != nil {
		return nil, fmt.Errorf("product repository fetchVariantOptionValueIDs: %w", err)
	}
	defer rows.Close()

	result := map[string][]string{}
	for rows.Next() {
		var variantID, valueID string
		if err := rows.Scan(&variantID, &valueID); err != nil {
			return nil, fmt.Errorf("product repository fetchVariantOptionValueIDs scan: %w", err)
		}
		result[variantID] = append(result[variantID], valueID)
	}
	return result, rows.Err()
}

func fetchVariantComponents(ctx context.Context, pool *pgxpool.Pool, variantIDs []string) (map[string][]domain.ProductVariantComponent, error) {
	rows, err := pool.Query(ctx, `
		SELECT pvc.id, pvc.parent_variant_id, pvc.quantity, pvc.role, cv.id, cv.mpn, cv.sku
		FROM product_variant_components pvc
		JOIN product_variants cv ON cv.id = pvc.component_variant_id
		WHERE pvc.parent_variant_id = ANY($1)
		ORDER BY pvc.role NULLS LAST`, variantIDs)
	if err != nil {
		return nil, fmt.Errorf("product repository fetchVariantComponents: %w", err)
	}
	defer rows.Close()

	result := map[string][]domain.ProductVariantComponent{}
	for rows.Next() {
		var id, parentID, compID, compMPN, compSKU string
		var quantity int
		var role *string
		if err := rows.Scan(&id, &parentID, &quantity, &role, &compID, &compMPN, &compSKU); err != nil {
			return nil, fmt.Errorf("product repository fetchVariantComponents scan: %w", err)
		}
		result[parentID] = append(result[parentID], domain.ProductVariantComponent{
			ID:        id,
			Component: domain.VariantRef{ID: compID, MPN: compMPN, SKU: compSKU},
			Quantity:  quantity,
			Role:      role,
		})
	}
	return result, rows.Err()
}

// scannedVariant is a scan-only intermediate, missing optionValueIDs/
// components — those are filled in by fetchVariantsForProduct's follow-up
// batched queries, same two-step shape scanProductWithRelationsRow uses for
// the specs/seo/images jsonb fields.
type scannedVariant struct {
	id, mpn, sku, stockStatus         string
	gtin                              *string
	isDefault, isStandalone, isActive bool
	leadTimeDays                      *int
	orderIndex                        int
	costPrice, salePrice              *int64
	originalPrice                     int64
	discountPercent                   float64
	weight                            *float64
	createdAt, updatedAt              time.Time
	deletedAt                         *time.Time
}

// attachDefaultVariants batch-loads just each product's DEFAULT variant (one
// row per product, no option-selections/components — those aren't needed
// for list-view display) and attaches it as the sole entry of Variants. This
// is what a list/grid read (GetAll) needs to show mpn/sku/gtin/price/stock
// without paying fetchVariantsForProduct's full-tree cost per product: a
// single extra query keyed by the already-cached default_variant_id column,
// not a join and not N+1. Products with no default_variant_id (shouldn't
// happen — every product has >=1 variant, see resolveDefaultVariant in the
// application layer) are simply left with a nil Variants slice.
func attachDefaultVariants(ctx context.Context, pool *pgxpool.Pool, products []*domain.ProductWithRelations) error {
	ids := make([]string, 0, len(products))
	for _, p := range products {
		if p.DefaultVariantID() != nil {
			ids = append(ids, *p.DefaultVariantID())
		}
	}
	if len(ids) == 0 {
		return nil
	}

	rows, err := pool.Query(ctx, `
		SELECT id, product_id, mpn, sku, gtin, is_default, is_standalone, stock_status::text, lead_time_days, cost_price,
		       original_price, sale_price, discount_percent, weight, is_active, order_index,
		       created_at, updated_at, deleted_at
		FROM product_variants
		WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("product repository attachDefaultVariants: %w", err)
	}
	defer rows.Close()

	byProductID := map[string]*domain.ProductVariant{}
	for rows.Next() {
		var v scannedVariant
		var productID string
		if err := rows.Scan(
			&v.id, &productID, &v.mpn, &v.sku, &v.gtin, &v.isDefault, &v.isStandalone, &v.stockStatus, &v.leadTimeDays, &v.costPrice,
			&v.originalPrice, &v.salePrice, &v.discountPercent, &v.weight, &v.isActive, &v.orderIndex,
			&v.createdAt, &v.updatedAt, &v.deletedAt,
		); err != nil {
			return fmt.Errorf("product repository attachDefaultVariants scan: %w", err)
		}
		byProductID[productID] = domain.RehydrateProductVariant(
			v.id, productID, v.mpn, v.sku, v.gtin,
			v.isDefault, v.isStandalone, v.stockStatus, v.leadTimeDays, v.costPrice,
			v.originalPrice, v.salePrice, v.discountPercent, v.weight, v.isActive, v.orderIndex,
			nil, nil,
			v.createdAt, v.updatedAt, v.deletedAt,
		)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("product repository attachDefaultVariants rows: %w", err)
	}

	for _, p := range products {
		if v, ok := byProductID[p.ID()]; ok {
			p.Variants = []*domain.ProductVariant{v}
		}
	}
	return nil
}

// attachVariantTree loads and attaches Options/Variants for a single-product
// read (GetByID/GetBySlug only — see fetchOptionsForProduct's doc comment
// for why this must never be called from a list/grid read path).
func (r *PostgresProductRepository) attachVariantTree(ctx context.Context, p *domain.ProductWithRelations) error {
	options, err := fetchOptionsForProduct(ctx, r.pool, p.ID())
	if err != nil {
		return err
	}
	variants, err := fetchVariantsForProduct(ctx, r.pool, p.ID())
	if err != nil {
		return err
	}
	p.Options = options
	p.Variants = variants
	return nil
}

// clearProductVariantTree deletes product_options (cascades to
// product_option_values and, via option_value's FK, product_variant_option_values)
// and product_variants (cascades to any remaining product_variant_option_values
// and to product_variant_components on both the parent and component side) —
// used by Update before writeProductVariantTree reinserts the whole tree. See
// domain/repository.go's ProductRepository.Update doc comment for why this is
// a wholesale replace, not an ID-matched upsert, for this pass.
func clearProductVariantTree(ctx context.Context, tx pgx.Tx, productID string) error {
	if _, err := tx.Exec(ctx, "DELETE FROM product_options WHERE product_id = $1", productID); err != nil {
		return fmt.Errorf("product repository clearVariantTree (options): %w", err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM product_variants WHERE product_id = $1", productID); err != nil {
		return fmt.Errorf("product repository clearVariantTree (variants): %w", err)
	}
	return nil
}

// writeProductVariantTree inserts options+values, then variants, then
// resolves each variant's OptionSelections (identified by option name/value,
// not ID — those IDs don't exist yet for a tree created in the same request)
// and Components (identified by index into the same variants slice) into the
// join/component tables. All within the caller's transaction.
func writeProductVariantTree(ctx context.Context, tx pgx.Tx, productID string, options []domain.ProductOptionInput, variants []domain.ProductVariantInput) error {
	// optionValueIDs["Màu sắc"]["Đỏ"] = <uuid>
	optionValueIDs := map[string]map[string]string{}

	for optIdx, opt := range options {
		var optionID string
		if err := tx.QueryRow(ctx,
			"INSERT INTO product_options (product_id, name, order_index) VALUES ($1,$2,$3) RETURNING id",
			productID, opt.Name, optIdx,
		).Scan(&optionID); err != nil {
			return fmt.Errorf("product repository writeVariantTree (insert option): %w", err)
		}
		optionValueIDs[opt.Name] = map[string]string{}
		for valIdx, val := range opt.Values {
			var valueID string
			if err := tx.QueryRow(ctx,
				"INSERT INTO product_option_values (option_id, value, order_index) VALUES ($1,$2,$3) RETURNING id",
				optionID, val, valIdx,
			).Scan(&valueID); err != nil {
				return fmt.Errorf("product repository writeVariantTree (insert option value): %w", err)
			}
			optionValueIDs[opt.Name][val] = valueID
		}
	}

	variantIDs := make([]string, len(variants))
	for i, v := range variants {
		sku := ""
		if v.SKU != nil {
			sku = *v.SKU
		}
		if sku == "" {
			sku = autoGenerateSKU(v.MPN)
		}
		stockStatus := v.StockStatus
		if stockStatus == "" {
			stockStatus = domain.StockStatusInStock
		}

		var variantID string
		err := tx.QueryRow(ctx, `
			INSERT INTO product_variants (
				product_id, mpn, sku, gtin, is_default, is_standalone, stock_status,
				lead_time_days, cost_price, original_price, sale_price, discount_percent,
				weight, is_active, order_index
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
			RETURNING id`,
			productID, v.MPN, sku, v.GTIN, v.IsDefault, !v.IsComponentOnly, stockStatus,
			v.LeadTimeDays, v.CostPrice, v.OriginalPrice, v.SalePrice, v.DiscountPercent,
			v.Weight, v.IsActive, i,
		).Scan(&variantID)
		if err != nil {
			return fmt.Errorf("product repository writeVariantTree (insert variant %q): %w", v.MPN, err)
		}
		variantIDs[i] = variantID

		for _, sel := range v.OptionSelections {
			values, ok := optionValueIDs[sel.OptionName]
			if !ok {
				return apperr.NewValidationError("validation failed", map[string][]string{
					"variants": {fmt.Sprintf("variant %q references unknown option %q", v.MPN, sel.OptionName)},
				})
			}
			valueID, ok := values[sel.Value]
			if !ok {
				return apperr.NewValidationError("validation failed", map[string][]string{
					"variants": {fmt.Sprintf("variant %q references unknown option value %q for option %q", v.MPN, sel.Value, sel.OptionName)},
				})
			}
			if _, err := tx.Exec(ctx,
				"INSERT INTO product_variant_option_values (variant_id, option_value_id) VALUES ($1,$2)",
				variantID, valueID,
			); err != nil {
				return fmt.Errorf("product repository writeVariantTree (link option value): %w", err)
			}
		}
	}

	for i, v := range variants {
		for _, comp := range v.Components {
			if comp.ComponentIndex < 0 || comp.ComponentIndex >= len(variantIDs) {
				return apperr.NewValidationError("validation failed", map[string][]string{
					"variants": {fmt.Sprintf("variant %q references an invalid component index %d", v.MPN, comp.ComponentIndex)},
				})
			}
			quantity := comp.Quantity
			if quantity <= 0 {
				quantity = 1
			}
			if _, err := tx.Exec(ctx,
				"INSERT INTO product_variant_components (parent_variant_id, component_variant_id, quantity, role) VALUES ($1,$2,$3,$4)",
				variantIDs[i], variantIDs[comp.ComponentIndex], quantity, comp.Role,
			); err != nil {
				return fmt.Errorf("product repository writeVariantTree (insert component): %w", err)
			}
		}
	}

	return nil
}

// autoGenerateSKU is used when the admin leaves a variant's sku blank —
// products_variants.sku is NOT NULL (internal warehouse code, always
// present), but unlike mpn it's fine for this business to not care what the
// value actually is. Not a primary key, so generating it in Go (rather than
// letting Postgres generate it) doesn't conflict with this project's
// "DB generates entity IDs" convention.
func autoGenerateSKU(mpn string) string {
	suffix := strings.ToUpper(uuid.NewString()[:8])
	prefix := strings.ToUpper(strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, mpn))
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	if prefix == "" {
		prefix = "VAR"
	}
	return prefix + "-" + suffix
}

// RecomputeDisplayCache recomputes products' denormalized read-cache columns
// (default_variant_id/display_price/display_stock_status/price_min/
// price_max/variant_mpns) from the current state of product_variants —
// called after every write to the variant tree, inside the same
// transaction. Always affects exactly one row (agg is a no-GROUP-BY
// aggregate, always 1 row even over an empty set; LEFT JOIN dv keeps that 1
// row even when there's no default variant), so a product with zero/
// no-longer-any variants correctly resets to NULL rather than keeping a
// stale cache — in practice this never happens for long, since the
// application layer (resolveDefaultVariant) rejects a create/update that
// would leave a product with zero variants. This is the same "compute once
// at write time, read via a plain column" technique already used for
// normalized_specs — see docs/product-v2-design.md.
func RecomputeDisplayCache(ctx context.Context, tx pgx.Tx, productID string) error {
	_, err := tx.Exec(ctx, `
		WITH dv AS (
			SELECT id, sale_price, original_price, stock_status
			FROM product_variants
			WHERE product_id = $1 AND deleted_at IS NULL AND is_default = true AND is_standalone = true
			LIMIT 1
		), agg AS (
			-- NULLIF(sale_price, 0): legacy data stores 0 (not NULL) to mean
			-- "no sale price". Without this, a variant with sale_price=0
			-- would incorrectly price_min/price_max to 0 instead of falling
			-- back to original_price. is_standalone = true excludes
			-- bundle-part-only variants (e.g. "Dàn nóng"/"Remote điều
			-- khiển", price 0, never independently sold, see
			-- product_variant_components) from the aggregate — without it,
			-- every bundled product's price_min collapses to 0.
			SELECT MIN(COALESCE(NULLIF(sale_price, 0), original_price)) AS price_min,
			       MAX(COALESCE(NULLIF(sale_price, 0), original_price)) AS price_max
			FROM product_variants
			WHERE product_id = $1 AND deleted_at IS NULL AND is_active = true AND is_standalone = true
		), mpns AS (
			-- Space-joined MPNs of every active variant (not just the
			-- default one) — feeds search_vector so a customer searching by
			-- ANY of a product's manufacturer codes finds it, not just the
			-- default variant's. sku is deliberately excluded: it's
			-- internal-only, not what customers search.
			SELECT string_agg(mpn, ' ' ORDER BY mpn) AS mpns
			FROM product_variants
			WHERE product_id = $1 AND deleted_at IS NULL AND is_active = true
		)
		UPDATE products p
		SET default_variant_id = dv.id,
		    display_price = COALESCE(NULLIF(dv.sale_price, 0), dv.original_price),
		    display_stock_status = dv.stock_status::text,
		    price_min = agg.price_min,
		    price_max = agg.price_max,
		    variant_mpns = COALESCE(mpns.mpns, '')
		FROM agg
		LEFT JOIN dv ON true
		LEFT JOIN mpns ON true
		WHERE p.id = $1`,
		productID,
	)
	if err != nil {
		return fmt.Errorf("product repository RecomputeDisplayCache: %w", err)
	}
	return nil
}
