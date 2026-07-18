package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/product/domain"
)

type PostgresProductRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ProductRepository = (*PostgresProductRepository)(nil)

func NewPostgresProductRepository(pool *pgxpool.Pool) *PostgresProductRepository {
	return &PostgresProductRepository{pool: pool}
}

// pgxQuerier is satisfied by both *pgxpool.Pool and pgx.Tx.
type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// fetchTagsForProducts batch-loads product_tags rows (+ tag name/slug) for
// every product id in one round trip, keyed by product id — same pattern as
// project's fetchCategoriesForProjects / news's fetchTagsForNews.
func fetchTagsForProducts(ctx context.Context, q pgxQuerier, productIDs []string) (map[string][]domain.TagRef, error) {
	if len(productIDs) == 0 {
		return map[string][]domain.TagRef{}, nil
	}

	query := `
		SELECT pt.product_id, t.id, t.name, t.slug
		FROM product_tags pt
		JOIN tags t ON t.id = pt.tag_id AND t.deleted_at IS NULL
		WHERE pt.product_id = ANY($1)
		ORDER BY t.name ASC`

	rows, err := q.Query(ctx, query, productIDs)
	if err != nil {
		return nil, fmt.Errorf("product repository fetchTags: %w", err)
	}
	defer rows.Close()

	result := map[string][]domain.TagRef{}
	for rows.Next() {
		var productID string
		var tag domain.TagRef
		if err := rows.Scan(&productID, &tag.ID, &tag.Name, &tag.Slug); err != nil {
			return nil, fmt.Errorf("product repository fetchTags scan: %w", err)
		}
		result[productID] = append(result[productID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("product repository fetchTags rows: %w", err)
	}
	return result, nil
}

func attachTagsToProducts(ctx context.Context, q pgxQuerier, products []*domain.ProductWithRelations) error {
	ids := make([]string, len(products))
	for i, p := range products {
		ids[i] = p.ID()
	}
	tagsByProduct, err := fetchTagsForProducts(ctx, q, ids)
	if err != nil {
		return err
	}
	for _, p := range products {
		p.Tags = tagsByProduct[p.ID()]
	}
	return nil
}

func insertProductTags(ctx context.Context, tx pgx.Tx, productID string, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO product_tags (product_id, tag_id) VALUES ")
	args := make([]any, 0, len(tagIDs)*2)
	argN := 1
	for i, tagID := range tagIDs {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("($%d, $%d)", argN, argN+1))
		args = append(args, productID, tagID)
		argN += 2
	}
	if _, err := tx.Exec(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("product repository insertTags: %w", err)
	}
	return nil
}

// productColumns/productJoin are shared by every read query that needs the
// joined category/brand display refs — GetAll, GetByID, GetBySlug, GetByIDs.
// LEFT JOIN (not INNER) on purpose: category_id/brand_id are NOT NULL on
// products, but a LEFT JOIN is defensive against a row whose referenced
// category/brand has been hard-deleted out from under it (shouldn't happen —
// category is ON DELETE RESTRICT, brand is ON DELETE SET NULL onto a NOT
// NULL column, i.e. physically blocked, see docs/brand.md — but costs
// nothing here and avoids silently dropping a product from list results if
// it ever does).
const productColumns = `p.id, p.category_id, p.brand_id, p.name, p.slug,
	p.description,
	p.images,
	p.is_featured, p.status, p.rejection_reason, p.order_index,
	p.meta_title, p.meta_description,
	p.product_line_id, p.short_description,
	p.default_variant_id, p.display_price, p.display_stock_status, p.price_min, p.price_max, p.variant_mpns,
	p.created_at, p.updated_at, p.deleted_at,
	c.id, c.name, c.slug, c.meta_title, c.meta_description,
	br.id, br.name, br.slug, br.logo_url, br.meta_title, br.meta_description, br.is_featured, br.order_index`

// productJoin only ever joins the cheap, always-useful category/brand refs —
// deliberately NEVER joined to product_variants/product_options here. List/
// grid reads (queryProducts, facets) must stay a single-table scan on
// products plus this join. List reads (GetAll/GetByIDs) separately
// batch-attach just each product's DEFAULT variant (see
// attachDefaultVariants in variant_repository.go) — enough for mpn/sku/
// gtin/price/stock display without the per-product cost of the full
// option/variant tree, which only single-product reads (GetByID/GetBySlug)
// load, via attachVariantTree. See docs/product-v2-design.md's performance
// section.
const productJoin = `FROM products p
	LEFT JOIN categories c ON c.id = p.category_id
	LEFT JOIN brands br ON br.id = p.brand_id`

// plainProductColumns is used for Create/Update's RETURNING clause — no
// joins, matches how brand/service's writes return a plain entity.
const plainProductColumns = `id, category_id, brand_id, name, slug, description,
	images,
	is_featured, status, rejection_reason, order_index,
	meta_title, meta_description,
	product_line_id, short_description,
	default_variant_id, display_price, display_stock_status, price_min, price_max, variant_mpns,
	created_at, updated_at, deleted_at`

// facetExclude tells buildFilterConditions to skip one filter dimension —
// used when computing that exact dimension's own facet counts, so e.g.
// switching brand stays visible as an option while one brand is currently
// selected (standard faceted-search "exclude own dimension" technique).
type facetExclude struct {
	Brand     bool
	Price     bool
	Attribute string // attribute code to exclude, "" = exclude none
}

// buildFilterConditions builds the dynamic WHERE clause the same way
// brand/service do (conditions []string + args []any + $N counter, never
// string-concatenating a value into the query). Covers scoping
// (category/brand/product-line/featured/status) plus search/price-range/
// attribute-facet dimensions, rebuilt on the structured attribute system —
// see domain.ProductFilter's doc comment.
func buildFilterConditions(filter domain.ProductFilter, exclude facetExclude) ([]string, []any) {
	conditions := []string{}
	args := []any{}
	argN := 1
	next := func() int {
		n := argN
		argN++
		return n
	}

	if !filter.IncludeDeleted {
		conditions = append(conditions, "p.deleted_at IS NULL")
	}

	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("p.category_id = $%d", next()))
		args = append(args, *filter.CategoryID)
	} else if len(filter.CategoryIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("p.category_id = ANY($%d::uuid[])", next()))
		args = append(args, filter.CategoryIDs)
	}

	if !exclude.Brand {
		switch {
		case filter.BrandID != nil:
			conditions = append(conditions, fmt.Sprintf("p.brand_id = $%d", next()))
			args = append(args, *filter.BrandID)
		case len(filter.BrandIDs) > 0:
			conditions = append(conditions, fmt.Sprintf("p.brand_id = ANY($%d::uuid[])", next()))
			args = append(args, filter.BrandIDs)
		}
	}

	if filter.ProductLineID != nil {
		conditions = append(conditions, fmt.Sprintf("p.product_line_id = $%d", next()))
		args = append(args, *filter.ProductLineID)
	}
	if filter.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("p.is_featured = $%d", next()))
		args = append(args, *filter.IsFeatured)
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("p.status = $%d", next()))
		args = append(args, *filter.Status)
	}

	if filter.Search != "" {
		// Combines full-text (accent/case-insensitive, name + variant MPNs)
		// with a trigram similarity fallback in one OR'd condition — catches
		// typos/partial matches without a separate zero-results probe query.
		n := next()
		conditions = append(conditions,
			fmt.Sprintf("(p.search_vector @@ websearch_to_tsquery('simple', immutable_unaccent($%d)) OR similarity(p.name, $%d) > 0.2)", n, n))
		args = append(args, filter.Search)
	}

	if !exclude.Price {
		if filter.MinPrice != nil {
			conditions = append(conditions, fmt.Sprintf("p.display_price >= $%d", next()))
			args = append(args, *filter.MinPrice)
		}
		if filter.MaxPrice != nil {
			conditions = append(conditions, fmt.Sprintf("p.display_price <= $%d", next()))
			args = append(args, *filter.MaxPrice)
		}
	}

	if len(filter.AttributeTokens) > 0 {
		byCode := map[string][]string{}
		for _, token := range filter.AttributeTokens {
			code, _, ok := strings.Cut(token, ":")
			if !ok || code == exclude.Attribute {
				continue
			}
			byCode[code] = append(byCode[code], token)
		}
		for _, tokens := range byCode {
			conditions = append(conditions, fmt.Sprintf("p.facet_tokens && $%d::text[]", next()))
			args = append(args, tokens)
		}
	}

	if len(filter.AttributeRanges) > 0 {
		for code, bounds := range filter.AttributeRanges {
			if code == exclude.Attribute {
				continue
			}
			rangeConds := []string{"ad2.code = $" + fmt.Sprint(next())}
			args = append(args, code)
			if bounds[0] != nil {
				rangeConds = append(rangeConds, fmt.Sprintf("pav.value_number >= $%d", next()))
				args = append(args, *bounds[0])
			}
			if bounds[1] != nil {
				rangeConds = append(rangeConds, fmt.Sprintf("pav.value_number <= $%d", next()))
				args = append(args, *bounds[1])
			}
			conditions = append(conditions, fmt.Sprintf(
				"EXISTS (SELECT 1 FROM product_attribute_values pav JOIN attribute_definitions ad2 ON ad2.id = pav.attribute_definition_id WHERE pav.product_id = p.id AND pav.deleted_at IS NULL AND %s)",
				strings.Join(rangeConds, " AND "),
			))
		}
	}

	return conditions, args
}

// sortClause maps ProductFilter.SortBy to an ORDER BY clause — "" keeps the
// pre-existing default (is_featured DESC, order_index ASC).
func sortClause(sortBy string) string {
	switch sortBy {
	case domain.SortByPriceAsc:
		return "ORDER BY p.display_price ASC NULLS LAST"
	case domain.SortByPriceDesc:
		return "ORDER BY p.display_price DESC NULLS LAST"
	case domain.SortByNewest:
		return "ORDER BY p.created_at DESC"
	default:
		return "ORDER BY p.is_featured DESC, p.order_index ASC"
	}
}

func (r *PostgresProductRepository) Count(ctx context.Context, filter domain.ProductFilter) (int, error) {
	conditions, args := buildFilterConditions(filter, facetExclude{})
	query := "SELECT COUNT(*) " + productJoin
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("product repository count: %w", err)
	}
	return count, nil
}

// GetAll runs the main list query, total count, and every facet dimension
// (brand/price/attribute) concurrently via errgroup — independent queries
// against the same pgxpool, each facet excluding its own filter dimension
// (see facetExclude).
func (r *PostgresProductRepository) GetAll(ctx context.Context, filter domain.ProductFilter) (*domain.ProductListResult, error) {
	var (
		products   []*domain.ProductWithRelations
		totalCount int
		facets     domain.ProductFacets
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		products, err = r.queryProducts(gctx, filter)
		return err
	})
	g.Go(func() error {
		conditions, args := buildFilterConditions(filter, facetExclude{})
		query := "SELECT COUNT(*) " + productJoin
		if len(conditions) > 0 {
			query += " WHERE " + strings.Join(conditions, " AND ")
		}
		return r.pool.QueryRow(gctx, query, args...).Scan(&totalCount)
	})
	g.Go(func() error {
		var err error
		facets.Brands, err = r.computeBrandFacets(gctx, filter)
		return err
	})
	g.Go(func() error {
		var err error
		facets.Price, err = r.computePriceFacets(gctx, filter)
		return err
	})
	g.Go(func() error {
		var err error
		facets.Attributes, err = r.computeAttributeFacets(gctx, filter)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("product repository getAll: %w", err)
	}

	if err := attachTagsToProducts(ctx, r.pool, products); err != nil {
		return nil, err
	}
	if err := attachDefaultVariants(ctx, r.pool, products); err != nil {
		return nil, err
	}

	return &domain.ProductListResult{Products: products, TotalCount: totalCount, Facets: facets}, nil
}

func (r *PostgresProductRepository) queryProducts(ctx context.Context, filter domain.ProductFilter) ([]*domain.ProductWithRelations, error) {
	conditions, args := buildFilterConditions(filter, facetExclude{})
	argN := len(args) + 1

	query := "SELECT " + productColumns + " " + productJoin
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " " + sortClause(filter.SortBy)

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, filter.Limit)
		argN++
	}
	query += fmt.Sprintf(" OFFSET $%d", argN)
	args = append(args, filter.Offset)
	argN++

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("product repository getAll query: %w", err)
	}
	defer rows.Close()

	var products []*domain.ProductWithRelations
	for rows.Next() {
		p, err := scanProductWithRelationsRow(rows)
		if err != nil {
			return nil, fmt.Errorf("product repository getAll scan: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("product repository getAll rows: %w", err)
	}

	return products, nil
}

func (r *PostgresProductRepository) GetByID(ctx context.Context, id string) (*domain.ProductWithRelations, error) {
	query := "SELECT " + productColumns + " " + productJoin + " WHERE p.id = $1 AND p.deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	p, err := scanProductWithRelationsRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("product repository getById: %w", err)
	}
	if err := attachTagsToProducts(ctx, r.pool, []*domain.ProductWithRelations{p}); err != nil {
		return nil, err
	}
	if err := r.attachVariantTree(ctx, p); err != nil {
		return nil, err
	}
	if err := attachAttributeValuesToProducts(ctx, r.pool, []*domain.ProductWithRelations{p}); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PostgresProductRepository) GetBySlug(ctx context.Context, slug string) (*domain.ProductWithRelations, error) {
	query := "SELECT " + productColumns + " " + productJoin + " WHERE p.slug = $1 AND p.deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	p, err := scanProductWithRelationsRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("product repository getBySlug: %w", err)
	}
	if err := attachTagsToProducts(ctx, r.pool, []*domain.ProductWithRelations{p}); err != nil {
		return nil, err
	}
	if err := r.attachVariantTree(ctx, p); err != nil {
		return nil, err
	}
	if err := attachAttributeValuesToProducts(ctx, r.pool, []*domain.ProductWithRelations{p}); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PostgresProductRepository) GetByIDs(ctx context.Context, ids []string) ([]*domain.ProductWithRelations, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := "SELECT " + productColumns + " " + productJoin + " WHERE p.id = ANY($1::uuid[]) AND p.deleted_at IS NULL"

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("product repository getByIds: %w", err)
	}
	defer rows.Close()

	var products []*domain.ProductWithRelations
	for rows.Next() {
		p, err := scanProductWithRelationsRow(rows)
		if err != nil {
			return nil, fmt.Errorf("product repository getByIds scan: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("product repository getByIds rows: %w", err)
	}
	if err := attachTagsToProducts(ctx, r.pool, products); err != nil {
		return nil, err
	}
	if err := attachDefaultVariants(ctx, r.pool, products); err != nil {
		return nil, err
	}
	return products, nil
}

// GetByIDsWithAttributeValues is GetByIDs plus attachAttributeValuesToProducts
// run across the whole batch — the first caller (Comparison) to pass more
// than one product through that function, which already batches via
// ANY($1) but was previously only ever invoked with single-element slices
// from GetByID/GetBySlug.
func (r *PostgresProductRepository) GetByIDsWithAttributeValues(ctx context.Context, ids []string) ([]*domain.ProductWithRelations, error) {
	products, err := r.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if err := attachAttributeValuesToProducts(ctx, r.pool, products); err != nil {
		return nil, err
	}
	return products, nil
}

// Create/Update marshal images by hand for the same reason brand's
// marshalFAQ/unmarshalFAQ do (see docs/brand.md and docs/catalog.md): the
// shared pool runs pgx.QueryExecModeSimpleProtocol (PgBouncer transaction
// pooler fix), which can't infer an OID for an arbitrary struct/slice —
// it must be marshaled to json.RawMessage by hand first. images is a plain
// text[]-backed column pgx encodes/decodes natively from/to []string with
// no special handling.
func (r *PostgresProductRepository) Create(ctx context.Context, product *domain.Product, tagIDs []string, options []domain.ProductOptionInput, variants []domain.ProductVariantInput, attributeValues []domain.ProductAttributeValueInput) (*domain.Product, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("product repository create (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO products (
			category_id, brand_id, name, slug, description,
			images,
			is_featured, status, rejection_reason, order_index,
			meta_title, meta_description,
			product_line_id, short_description
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING ` + plainProductColumns

	imagesJSON, err := media.MarshalImages(product.Images())
	if err != nil {
		return nil, fmt.Errorf("product repository create (marshal images): %w", err)
	}

	row := tx.QueryRow(ctx, query,
		product.CategoryID(), product.BrandID(), product.Name(), product.Slug(),
		orEmptyJSON(product.Description()),
		imagesJSON,
		product.IsFeatured(), string(product.Status()), product.RejectionReason(), product.OrderIndex(),
		product.MetaTitle(), product.MetaDescription(),
		product.ProductLineID(), product.ShortDescription(),
	)
	created, err := scanProduct(row)
	if err != nil {
		return nil, fmt.Errorf("product repository create: %w", err)
	}

	if err := insertProductTags(ctx, tx, created.ID(), tagIDs); err != nil {
		return nil, err
	}

	if err := writeProductVariantTree(ctx, tx, created.ID(), options, variants); err != nil {
		return nil, err
	}
	if err := RecomputeDisplayCache(ctx, tx, created.ID()); err != nil {
		return nil, err
	}
	if err := insertProductAttributeValues(ctx, tx, created.ID(), attributeValues); err != nil {
		return nil, err
	}
	if err := RecomputeFacetTokens(ctx, tx, created.ID()); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("product repository create (commit tx): %w", err)
	}

	// Re-scan so the returned entity reflects the cache columns
	// RecomputeDisplayCache just wrote (created still has them nil/zero from
	// the RETURNING clause above, which ran before variants existed).
	return r.getPlainByID(ctx, created.ID())
}

// Update omits updated_at from the SET list on purpose — the pre-existing
// update_products_modtime BEFORE UPDATE trigger sets it automatically on
// every UPDATE (unlike brand's Update, written before that trigger's
// presence was confirmed on this table). See docs/catalog.md.
func (r *PostgresProductRepository) Update(ctx context.Context, product *domain.Product, tagIDs *[]string, options *[]domain.ProductOptionInput, variants *[]domain.ProductVariantInput, attributeValues *[]domain.ProductAttributeValueInput) (*domain.Product, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("product repository update (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE products
		SET category_id = $1, brand_id = $2, name = $3, slug = $4,
			description = $5,
			images = $6,
			is_featured = $7, status = $8, rejection_reason = $9, order_index = $10,
			meta_title = $11, meta_description = $12,
			product_line_id = $13, short_description = $14
		WHERE id = $15
		RETURNING ` + plainProductColumns

	imagesJSON, err := media.MarshalImages(product.Images())
	if err != nil {
		return nil, fmt.Errorf("product repository update (marshal images): %w", err)
	}

	row := tx.QueryRow(ctx, query,
		product.CategoryID(), product.BrandID(), product.Name(), product.Slug(),
		orEmptyJSON(product.Description()),
		imagesJSON,
		product.IsFeatured(), string(product.Status()), product.RejectionReason(), product.OrderIndex(),
		product.MetaTitle(), product.MetaDescription(),
		product.ProductLineID(), product.ShortDescription(),
		product.ID(),
	)
	updated, err := scanProduct(row)
	if err != nil {
		return nil, fmt.Errorf("product repository update: %w", err)
	}

	if tagIDs != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM product_tags WHERE product_id = $1", updated.ID()); err != nil {
			return nil, fmt.Errorf("product repository update (clear tags): %w", err)
		}
		if err := insertProductTags(ctx, tx, updated.ID(), *tagIDs); err != nil {
			return nil, err
		}
	}

	// options/variants: nil means "leave the variant tree untouched" — same
	// convention as tagIDs. Non-nil replaces the whole tree wholesale
	// (delete-then-reinsert options/variants/components together), same
	// convention project-type already uses for project_type_category. This
	// means variant IDs are NOT stable across an update that touches the
	// variant tree — acceptable for this pass since nothing yet references
	// product_variants.id across requests (no order/pricing module exists
	// yet, see docs/product-v2-design.md); revisit if/when one does.
	if variants != nil {
		var opts []domain.ProductOptionInput
		if options != nil {
			opts = *options
		}
		if err := clearProductVariantTree(ctx, tx, updated.ID()); err != nil {
			return nil, err
		}
		if err := writeProductVariantTree(ctx, tx, updated.ID(), opts, *variants); err != nil {
			return nil, err
		}
		if err := RecomputeDisplayCache(ctx, tx, updated.ID()); err != nil {
			return nil, err
		}
	}

	// attributeValues: nil means "leave untouched" — same convention as
	// tagIDs/options/variants.
	if attributeValues != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM product_attribute_values WHERE product_id = $1", updated.ID()); err != nil {
			return nil, fmt.Errorf("product repository update (clear attribute values): %w", err)
		}
		if err := insertProductAttributeValues(ctx, tx, updated.ID(), *attributeValues); err != nil {
			return nil, err
		}
		if err := RecomputeFacetTokens(ctx, tx, updated.ID()); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("product repository update (commit tx): %w", err)
	}
	if variants != nil {
		return r.getPlainByID(ctx, updated.ID())
	}
	return updated, nil
}

// getPlainByID re-reads a product's plain columns after a variant-tree write
// so the returned entity reflects the just-recomputed display cache —
// RETURNING on the INSERT/UPDATE above ran before the cache existed.
func (r *PostgresProductRepository) getPlainByID(ctx context.Context, id string) (*domain.Product, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+plainProductColumns+" FROM products WHERE id = $1", id)
	p, err := scanProduct(row)
	if err != nil {
		return nil, fmt.Errorf("product repository getPlainByID: %w", err)
	}
	return p, nil
}

func (r *PostgresProductRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE products SET deleted_at = $1 WHERE id = $2", time.Now(), id)
	if err != nil {
		return fmt.Errorf("product repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresProductRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE products SET deleted_at = NULL WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("product repository restore: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

// orEmptyJSON defaults a nil/empty description to "{}" — products.description
// is NOT NULL DEFAULT '{}' jsonb; an explicit NULL bind would violate the
// NOT NULL constraint (the column default only applies when a column is
// omitted from the INSERT list entirely, not when NULL is given explicitly).
func orEmptyJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	return raw
}

// orEmptyStrings defaults a nil slice to an empty (non-nil) one before
// binding to a text[] column — matches the column's own DEFAULT '{}' intent
// rather than writing an explicit SQL NULL for "no items".
func orEmptyStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func scanProduct(row rowScanner) (*domain.Product, error) {
	var (
		id, categoryID, brandID, name, slug  string
		description                          json.RawMessage
		imagesRaw                            []byte
		isFeatured                           bool
		status                               string
		rejectionReason                      *string
		orderIndex                           int
		metaTitle, metaDescription           *string
		productLineID, shortDescription      *string
		defaultVariantID, displayStockStatus *string
		displayPrice, priceMin, priceMax     *int64
		variantMpns                          string
		createdAt, updatedAt                 time.Time
		deletedAt                            *time.Time
	)

	if err := row.Scan(
		&id, &categoryID, &brandID, &name, &slug,
		&description,
		&imagesRaw,
		&isFeatured, &status, &rejectionReason, &orderIndex,
		&metaTitle, &metaDescription,
		&productLineID, &shortDescription,
		&defaultVariantID, &displayPrice, &displayStockStatus, &priceMin, &priceMax, &variantMpns,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	images, err := media.UnmarshalImages(imagesRaw)
	if err != nil {
		return nil, fmt.Errorf("scan product (unmarshal images): %w", err)
	}

	return domain.RehydrateProduct(
		id, categoryID, brandID, name, slug,
		description,
		images,
		isFeatured, domain.ProductStatus(status), rejectionReason, orderIndex,
		metaTitle, metaDescription,
		productLineID, shortDescription,
		defaultVariantID, displayPrice, displayStockStatus, priceMin, priceMax, variantMpns,
		createdAt, updatedAt, deletedAt,
	), nil
}

// scanProductWithRelationsRow scans the productColumns + productJoin shape.
func scanProductWithRelationsRow(row rowScanner) (*domain.ProductWithRelations, error) {
	var (
		id, categoryID, brandID, name, slug  string
		description                          json.RawMessage
		imagesRaw                            []byte
		isFeatured                           bool
		status                               string
		rejectionReason                      *string
		orderIndex                           int
		metaTitle, metaDescription           *string
		productLineID, shortDescription      *string
		defaultVariantID, displayStockStatus *string
		displayPrice, priceMin, priceMax     *int64
		variantMpns                          string
		createdAt, updatedAt                 time.Time
		deletedAt                            *time.Time

		catID, catName, catSlug, catMetaTitle, catMetaDescription *string

		brID, brName, brSlug, brLogoURL, brMetaTitle, brMetaDescription *string
		brIsFeatured                                                    *bool
		brOrderIndex                                                    *int
	)

	dest := []any{
		&id, &categoryID, &brandID, &name, &slug,
		&description,
		&imagesRaw,
		&isFeatured, &status, &rejectionReason, &orderIndex,
		&metaTitle, &metaDescription,
		&productLineID, &shortDescription,
		&defaultVariantID, &displayPrice, &displayStockStatus, &priceMin, &priceMax, &variantMpns,
		&createdAt, &updatedAt, &deletedAt,
		&catID, &catName, &catSlug, &catMetaTitle, &catMetaDescription,
		&brID, &brName, &brSlug, &brLogoURL, &brMetaTitle, &brMetaDescription, &brIsFeatured, &brOrderIndex,
	}

	if err := row.Scan(dest...); err != nil {
		return nil, err
	}

	images, err := media.UnmarshalImages(imagesRaw)
	if err != nil {
		return nil, fmt.Errorf("scan product with relations (unmarshal images): %w", err)
	}

	product := domain.RehydrateProduct(
		id, categoryID, brandID, name, slug,
		description,
		images,
		isFeatured, domain.ProductStatus(status), rejectionReason, orderIndex,
		metaTitle, metaDescription,
		productLineID, shortDescription,
		defaultVariantID, displayPrice, displayStockStatus, priceMin, priceMax, variantMpns,
		createdAt, updatedAt, deletedAt,
	)

	result := &domain.ProductWithRelations{Product: product}
	if catID != nil {
		result.Category = &domain.CategoryRef{
			ID: *catID, Name: derefStr(catName), Slug: derefStr(catSlug),
			MetaTitle: catMetaTitle, MetaDescription: catMetaDescription,
		}
	}
	if brID != nil {
		result.Brand = &domain.BrandRef{
			ID: *brID, Name: derefStr(brName), Slug: derefStr(brSlug), LogoURL: derefStr(brLogoURL),
			MetaTitle: brMetaTitle, MetaDescription: brMetaDescription,
			IsFeatured: brIsFeatured != nil && *brIsFeatured,
			OrderIndex: derefInt(brOrderIndex),
		}
	}

	return result, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
