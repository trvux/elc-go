package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/trvux/elc-go/internal/catalog/domain"
)

type PostgresProductRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ProductRepository = (*PostgresProductRepository)(nil)

func NewPostgresProductRepository(pool *pgxpool.Pool) *PostgresProductRepository {
	return &PostgresProductRepository{pool: pool}
}

// productColumns/catalogJoin are shared by every read query that needs the
// joined category/brand display refs — GetAll, GetByID, GetBySlug, GetByIDs.
// LEFT JOIN (not INNER) on purpose: category_id/brand_id are NOT NULL on
// products, but a LEFT JOIN is defensive against a row whose referenced
// category/brand has been hard-deleted out from under it (shouldn't happen —
// category is ON DELETE RESTRICT, brand is ON DELETE SET NULL onto a NOT
// NULL column, i.e. physically blocked, see docs/brand.md — but costs
// nothing here and avoids silently dropping a product from list results if
// it ever does).
const productColumns = `p.id, p.category_id, p.brand_id, p.name, p.sku, p.slug,
	p.description, p.specs, p.normalized_specs,
	p.images, p.labels,
	p.original_price, p.sale_price, p.discount_percent,
	p.is_featured, p.is_published, p.order_index,
	p.stock_status, p.condition,
	p.meta_title, p.meta_description, p.mpn, p.gtin,
	p.created_at, p.updated_at, p.deleted_at,
	c.id, c.name, c.slug, c.meta_title, c.meta_description,
	br.id, br.name, br.slug, br.logo_url, br.meta_title, br.meta_description, br.is_featured, br.order_index`

const catalogJoin = `FROM products p
	LEFT JOIN categories c ON c.id = p.category_id
	LEFT JOIN brands br ON br.id = p.brand_id`

// plainProductColumns is used for Create/Update's RETURNING clause — no
// joins, matches how brand/service's writes return a plain entity.
const plainProductColumns = `id, category_id, brand_id, name, sku, slug, description, specs, normalized_specs,
	images, labels, original_price, sale_price, discount_percent,
	is_featured, is_published, order_index, stock_status, condition,
	meta_title, meta_description, mpn, gtin, created_at, updated_at, deleted_at`

// categoryPriorityOrder is the exact "popularity" sort priority list from the
// old TS CATEGORY_PRIORITY_ORDER in searchProducts.ts — order is
// load-bearing, do not re-sort or "clean up".
var categoryPriorityOrder = []string{
	"may-lanh-treo-tuong",
	"may-loc-khong-khi-may-loc-nuoc",
	"may-lanh-dieu-hoa-tu-dung",
	"may-lanh-am-tran",
	"may-lanh-giau-tran-noi-ong-gio",
	"may-lanh-ap-tran",
	"may-loc-khong-khi-may-cap-khi-tuoi-loc-khong-khi",
	"may-loc-khong-khi-phu-kien-dong-bo-cua-he-thong-cap-gio-tuoi",
}

// popularityOrderBy is built once from categoryPriorityOrder — the slugs are
// a fixed compile-time list (never user input), so building this via
// fmt.Sprintf instead of bind parameters is safe and avoids 8 extra
// positional args on every single list query.
var popularityOrderBy = buildPopularityOrderBy()

func buildPopularityOrderBy() string {
	var b strings.Builder
	b.WriteString("CASE c.slug ")
	for i, slug := range categoryPriorityOrder {
		fmt.Fprintf(&b, "WHEN '%s' THEN %d ", slug, i)
	}
	b.WriteString("ELSE 999 END, p.is_featured DESC, p.order_index ASC")
	return b.String()
}

func buildOrderBy(filter domain.ProductFilter) string {
	switch domain.ProductSortBy(filter.SortBy) {
	case domain.SortByPriceAsc:
		return "COALESCE(p.sale_price, p.original_price) ASC"
	case domain.SortByPriceDesc:
		return "COALESCE(p.sale_price, p.original_price) DESC"
	case domain.SortByNewest:
		return "p.created_at DESC"
	case domain.SortByDiscountDesc:
		return "p.discount_percent DESC"
	default:
		// "popularity" and anything unrecognized fall back to the same
		// default the old TS code used.
		return popularityOrderBy
	}
}

// fuzzyWordSimilarityThreshold is the per-token pg_trgm word_similarity cutoff
// for the typo-tolerance search fallback (see buildFilterConditions' search
// clause for the empirical reasoning behind per-token AND matching). 0.35
// comfortably passes a real one-edit typo ("inveter" vs "Inverter" scores
// 0.545) while still requiring genuine resemblance, not just a shared short
// substring.
const fuzzyWordSimilarityThreshold = 0.35

// searchTokens splits a search query into words for the per-token fuzzy
// match — same tokenization spirit as the old TS getQueryTokens/tokenize
// (whitespace-separated), simplified since Postgres's word_similarity
// already handles case/accent normalization once combined with
// immutable_unaccent at the call site, so no lowercasing is needed here.
func searchTokens(q string) []string {
	fields := strings.Fields(q)
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			tokens = append(tokens, f)
		}
	}
	return tokens
}

type filterExclude int

const (
	excludeNone filterExclude = iota
	excludeBrand
	excludeSpecs
	excludePrice
)

// buildFilterConditions builds the dynamic WHERE clause the same way
// brand/service do (conditions []string + args []any + $N counter, never
// string-concatenating a value into the query). exclude skips one filter
// dimension — used by GetAll's facet queries, which must ignore the very
// dimension they're faceting on (standard faceted-search technique: the
// brand facet counts should reflect every OTHER active filter but not the
// brand filter itself, so switching brands stays visible as an option).
//
// fuzzy selects which of the two mutually-exclusive search strategies to
// apply when filter.Search != "" — see resolveSearchMode for how this is
// decided and why full-text and fuzzy are never OR'd together in the same
// query (empirically, blending them let a fuzzy word-level false positive
// smuggle in results a plain full-text match would never have returned).
func buildFilterConditions(filter domain.ProductFilter, exclude filterExclude, fuzzy bool) ([]string, []any) {
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

	if exclude != excludeBrand {
		switch {
		case filter.BrandID != nil:
			conditions = append(conditions, fmt.Sprintf("p.brand_id = $%d", next()))
			args = append(args, *filter.BrandID)
		case len(filter.BrandIDs) > 0:
			conditions = append(conditions, fmt.Sprintf("p.brand_id = ANY($%d::uuid[])", next()))
			args = append(args, filter.BrandIDs)
		case len(filter.BrandSlugs) > 0:
			conditions = append(conditions, fmt.Sprintf("br.slug = ANY($%d::text[])", next()))
			args = append(args, filter.BrandSlugs)
		}
	}

	if filter.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("p.is_featured = $%d", next()))
		args = append(args, *filter.IsFeatured)
	}
	if filter.IsPublished != nil {
		conditions = append(conditions, fmt.Sprintf("p.is_published = $%d", next()))
		args = append(args, *filter.IsPublished)
	}
	if filter.Condition != "" {
		conditions = append(conditions, fmt.Sprintf("p.condition = $%d", next()))
		args = append(args, filter.Condition)
	}

	if exclude != excludePrice {
		if filter.MinPrice != nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(p.sale_price, p.original_price) >= $%d", next()))
			args = append(args, *filter.MinPrice)
		}
		if filter.MaxPrice != nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(p.sale_price, p.original_price) <= $%d", next()))
			args = append(args, *filter.MaxPrice)
		}
	}

	// Specs filter: AND-across-label-groups, OR-within-a-label-group's
	// values — a product matches a (label, values) pair if it has ANY of
	// those normalized_specs entries (array overlap, &&), and it must match
	// EVERY requested label group (one && condition per label, ANDed
	// together via `conditions`). Sorted keys purely for deterministic SQL
	// generation (easier to reason about/log), not a correctness requirement.
	if exclude != excludeSpecs && len(filter.Specs) > 0 {
		labels := make([]string, 0, len(filter.Specs))
		for label := range filter.Specs {
			labels = append(labels, label)
		}
		sort.Strings(labels)
		for _, label := range labels {
			values := filter.Specs[label]
			if len(values) == 0 {
				continue
			}
			formatted := make([]string, len(values))
			for i, v := range values {
				formatted[i] = label + "::" + v
			}
			conditions = append(conditions, fmt.Sprintf("p.normalized_specs && $%d::text[]", next()))
			args = append(args, formatted)
		}
	}

	// Search: two mutually exclusive strategies, never blended in the same
	// query — see resolveSearchMode for why. fuzzy==false: plain full-text
	// (search_vector, name+sku via 'simple' config + unaccent) — the
	// high-precision default, used whenever it finds anything at all.
	// fuzzy==true: PER-TOKEN pg_trgm word_similarity with AND across tokens
	// (a typo-tolerance fallback, only reached when full-text found nothing)
	// — comparing the whole query string against the whole name in one shot
	// was tried and rejected: the whole-string query "am tran" (unaccented
	// "âm trần", a ceiling-cassette AC line) scored word_similarity 0.625
	// against the whole name of a DIFFERENT, unrelated product line, "Máy
	// lạnh giấu trần..." ("giấu trần" = duct concealed-ceiling) — *higher*
	// than the legitimate "inveter"→"Inverter" typo match's 0.545 — because
	// "trần" alone overlaps strongly even though "âm" shares nothing with
	// "giấu". No single scalar threshold separates those two cases; per-
	// token AND does: "giấu trần" scores 0 on the "am" token (no word in it
	// resembles "am") and is excluded, while "âm trần" scores 1.0 on both
	// tokens and is included. Same per-token AND semantics the old Fuse.js
	// search used (`queryTokens.every(set => set.has(id))`), pushed into SQL.
	// See docs/catalog.md.
	//
	// immutable_unaccent (defined in the catalog migration) is used here
	// instead of plain unaccent() purely for consistency with the
	// search_vector generated column, which had to use the IMMUTABLE wrapper
	// (see the migration's comment) — both call the same 'unaccent'
	// dictionary either way. See docs/catalog.md for why this search
	// combination and not a dedicated search engine.
	if filter.Search != "" {
		if !fuzzy {
			conditions = append(conditions, fmt.Sprintf(
				"p.search_vector @@ websearch_to_tsquery('simple', immutable_unaccent($%d))", next(),
			))
			args = append(args, filter.Search)
		} else if tokens := searchTokens(filter.Search); len(tokens) > 0 {
			fuzzyParts := make([]string, len(tokens))
			for i, tok := range tokens {
				fuzzyParts[i] = fmt.Sprintf(
					"word_similarity(immutable_unaccent($%d), immutable_unaccent(p.name)) > %g",
					next(), fuzzyWordSimilarityThreshold,
				)
				args = append(args, tok)
			}
			conditions = append(conditions, "("+strings.Join(fuzzyParts, " AND ")+")")
		} else {
			// Search set but tokenizes to nothing (e.g. only punctuation) —
			// match nothing rather than silently ignoring the filter.
			conditions = append(conditions, "false")
		}
	}

	return conditions, args
}

// resolveSearchMode decides which of the two search strategies
// buildFilterConditions should use: plain full-text is tried first (highest
// precision), and the per-token fuzzy fallback only kicks in when full-text,
// combined with every OTHER currently-active filter, finds nothing at all —
// never blended together in one query. See buildFilterConditions' search
// comment for the false-positive this was designed to avoid.
func (r *PostgresProductRepository) resolveSearchMode(ctx context.Context, filter domain.ProductFilter) (fuzzy bool, err error) {
	if filter.Search == "" {
		return false, nil
	}

	conditions, args := buildFilterConditions(filter, excludeNone, false)
	query := "SELECT EXISTS(SELECT 1 " + catalogJoin
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += ")"

	var exists bool
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("product repository resolveSearchMode: %w", err)
	}
	return !exists, nil
}

func (r *PostgresProductRepository) Count(ctx context.Context, filter domain.ProductFilter) (int, error) {
	fuzzy, err := r.resolveSearchMode(ctx, filter)
	if err != nil {
		return 0, err
	}

	conditions, args := buildFilterConditions(filter, excludeNone, fuzzy)
	query := "SELECT COUNT(*) " + catalogJoin
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("product repository count: %w", err)
	}
	return count, nil
}

// GetAll runs the main list query plus the total count and all three facet
// aggregates concurrently via errgroup — five independent queries against
// the same pgxpool, each on its own pooled connection. This is the "efficient
// single round trip per concern" version of what the old TS searchProducts.ts
// did sequentially in-process against an already-fully-loaded product list.
//
// The search mode (plain full-text vs. fuzzy fallback, see resolveSearchMode)
// is resolved ONCE up front and shared by every one of the five queries below
// — they must all agree on the same mode, otherwise e.g. the brand facet
// could be computed against a different result set than the product list
// it's supposed to describe.
func (r *PostgresProductRepository) GetAll(ctx context.Context, filter domain.ProductFilter) (*domain.ProductListResult, error) {
	fuzzy, err := r.resolveSearchMode(ctx, filter)
	if err != nil {
		return nil, err
	}

	var (
		products   []*domain.ProductWithRelations
		totalCount int
		facets     domain.ProductFacets
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		products, err = r.queryProducts(gctx, filter, fuzzy)
		return err
	})
	g.Go(func() error {
		conditions, args := buildFilterConditions(filter, excludeNone, fuzzy)
		query := "SELECT COUNT(*) " + catalogJoin
		if len(conditions) > 0 {
			query += " WHERE " + strings.Join(conditions, " AND ")
		}
		return r.pool.QueryRow(gctx, query, args...).Scan(&totalCount)
	})
	g.Go(func() error {
		var err error
		facets.Brands, err = r.brandFacets(gctx, filter, fuzzy)
		return err
	})
	g.Go(func() error {
		var err error
		facets.Specs, err = r.specFacets(gctx, filter, fuzzy)
		return err
	})
	g.Go(func() error {
		var err error
		facets.MinPrice, facets.MaxPrice, err = r.priceFacets(gctx, filter, fuzzy)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("product repository getAll: %w", err)
	}

	return &domain.ProductListResult{Products: products, TotalCount: totalCount, Facets: facets}, nil
}

func (r *PostgresProductRepository) queryProducts(ctx context.Context, filter domain.ProductFilter, fuzzy bool) ([]*domain.ProductWithRelations, error) {
	conditions, args := buildFilterConditions(filter, excludeNone, fuzzy)
	argN := len(args) + 1

	// The search_rank column is always computed (even when not searching, in
	// which case it's just 0 for every row) so every row of this query has
	// the same column shape regardless of whether filter.Search is set —
	// simpler than branching the SELECT list and Scan call. At 196 rows the
	// wasted computation when not searching is negligible; see docs/catalog.md.
	//
	// Rank matches whichever single strategy buildFilterConditions actually
	// filtered with — never both, same reasoning as the search condition
	// itself. fuzzy==true rows didn't match full-text at all (that's why
	// fuzzy mode was reached), so ts_rank would just be 0 noise for all of
	// them; fuzzy==false rows all matched full-text, so word_similarity
	// isn't needed to rank them.
	var rankExpr string
	if !fuzzy {
		rankExpr = fmt.Sprintf(
			"COALESCE(ts_rank(p.search_vector, websearch_to_tsquery('simple', immutable_unaccent($%d))), 0) AS search_rank",
			argN,
		)
		args = append(args, filter.Search)
		argN++
	} else if tokens := searchTokens(filter.Search); len(tokens) > 0 {
		// Average (not min/max) per-token word_similarity — rewards rows
		// where every token matches well without letting one strong token
		// alone dominate the score.
		parts := make([]string, len(tokens))
		for i, tok := range tokens {
			parts[i] = fmt.Sprintf(
				"COALESCE(word_similarity(immutable_unaccent($%d), immutable_unaccent(p.name)), 0)",
				argN,
			)
			args = append(args, tok)
			argN++
		}
		rankExpr = fmt.Sprintf("((%s) / %d) AS search_rank", strings.Join(parts, " + "), len(tokens))
	} else {
		rankExpr = "0 AS search_rank"
	}

	query := "SELECT " + productColumns + ", " + rankExpr + " " + catalogJoin
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	orderBy := buildOrderBy(filter)
	if filter.Search != "" {
		// When actively searching, rank wins first; the requested/default
		// sort is only a tiebreaker among equally-ranked results.
		orderBy = "search_rank DESC, " + orderBy
	}
	query += " ORDER BY " + orderBy

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
		var searchRank float64
		p, err := scanProductWithRelationsRow(rows, &searchRank)
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

// brandFacets ignores the brand dimension of the current filter (exclude ==
// excludeBrand) so it reports every brand available under the OTHER active
// filters — the standard faceted-search technique, letting a user see (and
// switch to) sibling brands while one is already selected.
func (r *PostgresProductRepository) brandFacets(ctx context.Context, filter domain.ProductFilter, fuzzy bool) ([]domain.BrandFacet, error) {
	conditions, args := buildFilterConditions(filter, excludeBrand, fuzzy)
	conditions = append(conditions, "br.id IS NOT NULL")

	query := "SELECT DISTINCT br.id, br.name, br.slug " + catalogJoin +
		" WHERE " + strings.Join(conditions, " AND ") +
		" ORDER BY br.name ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("product repository brandFacets: %w", err)
	}
	defer rows.Close()

	var facets []domain.BrandFacet
	for rows.Next() {
		var f domain.BrandFacet
		if err := rows.Scan(&f.ID, &f.Name, &f.Slug); err != nil {
			return nil, fmt.Errorf("product repository brandFacets scan: %w", err)
		}
		facets = append(facets, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("product repository brandFacets rows: %w", err)
	}
	return facets, nil
}

// specFacets ignores the specs dimension for the same reason brandFacets
// ignores brand. unnest() flattens each matching row's normalized_specs
// array into one row per "Label::Value" entry; GROUP BY collapses duplicates
// (a facet value only needs to be listed once, regardless of how many
// products have it) — the grouping itself IS the "count > 0" filter, there's
// no separate HAVING needed.
func (r *PostgresProductRepository) specFacets(ctx context.Context, filter domain.ProductFilter, fuzzy bool) ([]domain.SpecFacet, error) {
	conditions, args := buildFilterConditions(filter, excludeSpecs, fuzzy)
	query := "SELECT unnest(p.normalized_specs) AS facet " + catalogJoin
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " GROUP BY facet"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("product repository specFacets: %w", err)
	}
	defer rows.Close()

	labelOrder := []string{}
	grouped := map[string][]string{}
	for rows.Next() {
		var facet string
		if err := rows.Scan(&facet); err != nil {
			return nil, fmt.Errorf("product repository specFacets scan: %w", err)
		}
		label, value, ok := strings.Cut(facet, "::")
		if !ok {
			continue
		}
		if _, exists := grouped[label]; !exists {
			labelOrder = append(labelOrder, label)
		}
		grouped[label] = append(grouped[label], value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("product repository specFacets rows: %w", err)
	}

	result := make([]domain.SpecFacet, 0, len(labelOrder))
	for _, label := range labelOrder {
		values := grouped[label]
		sort.Strings(values)
		result = append(result, domain.SpecFacet{Label: label, Values: values})
	}
	return result, nil
}

func (r *PostgresProductRepository) priceFacets(ctx context.Context, filter domain.ProductFilter, fuzzy bool) (int64, int64, error) {
	conditions, args := buildFilterConditions(filter, excludePrice, fuzzy)
	query := "SELECT MIN(COALESCE(p.sale_price, p.original_price)), MAX(COALESCE(p.sale_price, p.original_price)) " + catalogJoin
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var minPrice, maxPrice *int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&minPrice, &maxPrice); err != nil {
		return 0, 0, fmt.Errorf("product repository priceFacets: %w", err)
	}

	var minVal, maxVal int64
	if minPrice != nil {
		minVal = *minPrice
	}
	if maxPrice != nil {
		maxVal = *maxPrice
	}
	return minVal, maxVal, nil
}

func (r *PostgresProductRepository) GetByID(ctx context.Context, id string) (*domain.ProductWithRelations, error) {
	query := "SELECT " + productColumns + " " + catalogJoin + " WHERE p.id = $1 AND p.deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	p, err := scanProductWithRelationsRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("product repository getById: %w", err)
	}
	return p, nil
}

func (r *PostgresProductRepository) GetBySlug(ctx context.Context, slug string) (*domain.ProductWithRelations, error) {
	query := "SELECT " + productColumns + " " + catalogJoin + " WHERE p.slug = $1 AND p.deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	p, err := scanProductWithRelationsRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("product repository getBySlug: %w", err)
	}
	return p, nil
}

func (r *PostgresProductRepository) GetByIDs(ctx context.Context, ids []string) ([]*domain.ProductWithRelations, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := "SELECT " + productColumns + " " + catalogJoin + " WHERE p.id = ANY($1::uuid[]) AND p.deleted_at IS NULL"

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
	return products, nil
}

// Create/Update marshal specs by hand for the same reason brand's
// marshalFAQ/unmarshalFAQ do (see docs/brand.md and docs/catalog.md): the
// shared pool runs pgx.QueryExecModeSimpleProtocol (PgBouncer transaction
// pooler fix), which can't infer an OID for an arbitrary []domain.SpecItem —
// it must be marshaled to json.RawMessage by hand first. images/labels/
// normalized_specs are plain text[] columns, which pgx encodes/decodes
// natively from/to []string with no special handling (same as
// service.labels).
func (r *PostgresProductRepository) Create(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	query := `
		INSERT INTO products (
			category_id, brand_id, name, sku, slug, description, specs, normalized_specs,
			images, labels, original_price, sale_price, discount_percent,
			is_featured, is_published, order_index, stock_status, condition,
			meta_title, meta_description, mpn, gtin
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
		RETURNING ` + plainProductColumns

	specsJSON, err := marshalSpecs(product.Specs())
	if err != nil {
		return nil, fmt.Errorf("product repository create (marshal specs): %w", err)
	}

	row := r.pool.QueryRow(ctx, query,
		product.CategoryID(), product.BrandID(), product.Name(), product.SKU(), product.Slug(),
		orEmptyJSON(product.Description()), specsJSON, orEmptyStrings(product.NormalizedSpecs()),
		orEmptyStrings(product.Images()), orEmptyStrings(product.Labels()),
		product.OriginalPrice(), product.SalePrice(), product.DiscountPercent(),
		product.IsFeatured(), product.IsPublished(), product.OrderIndex(),
		product.StockStatus(), product.Condition(),
		product.MetaTitle(), product.MetaDescription(), product.MPN(), product.GTIN(),
	)
	created, err := scanProduct(row)
	if err != nil {
		return nil, fmt.Errorf("product repository create: %w", err)
	}
	return created, nil
}

// Update omits updated_at from the SET list on purpose — the pre-existing
// update_products_modtime BEFORE UPDATE trigger sets it automatically on
// every UPDATE (unlike brand's Update, written before that trigger's
// presence was confirmed on this table). See docs/catalog.md.
func (r *PostgresProductRepository) Update(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	query := `
		UPDATE products
		SET category_id = $1, brand_id = $2, name = $3, sku = $4, slug = $5,
			description = $6, specs = $7, normalized_specs = $8,
			images = $9, labels = $10, original_price = $11, sale_price = $12, discount_percent = $13,
			is_featured = $14, is_published = $15, order_index = $16, stock_status = $17, condition = $18,
			meta_title = $19, meta_description = $20, mpn = $21, gtin = $22
		WHERE id = $23
		RETURNING ` + plainProductColumns

	specsJSON, err := marshalSpecs(product.Specs())
	if err != nil {
		return nil, fmt.Errorf("product repository update (marshal specs): %w", err)
	}

	row := r.pool.QueryRow(ctx, query,
		product.CategoryID(), product.BrandID(), product.Name(), product.SKU(), product.Slug(),
		orEmptyJSON(product.Description()), specsJSON, orEmptyStrings(product.NormalizedSpecs()),
		orEmptyStrings(product.Images()), orEmptyStrings(product.Labels()),
		product.OriginalPrice(), product.SalePrice(), product.DiscountPercent(),
		product.IsFeatured(), product.IsPublished(), product.OrderIndex(),
		product.StockStatus(), product.Condition(),
		product.MetaTitle(), product.MetaDescription(), product.MPN(), product.GTIN(),
		product.ID(),
	)
	updated, err := scanProduct(row)
	if err != nil {
		return nil, fmt.Errorf("product repository update: %w", err)
	}
	return updated, nil
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

type adjacentRow struct {
	id, name, slug string
}

// GetAdjacent implements the "same-category siblings first, fall back to the
// full published catalog when fewer than 2 published siblings exist" rule
// from the old getAdjacentProducts.ts, as two indexed queries at most (the
// fallback only ever runs when the first query came back too small) instead
// of the two full-repository-round-trip calls the old TS application layer
// made — see application/get_adjacent_products.go for why this was pushed
// down here.
func (r *PostgresProductRepository) GetAdjacent(ctx context.Context, categoryID, currentID string) (*domain.AdjacentProduct, *domain.AdjacentProduct, error) {
	siblings, err := r.fetchOrderedPublished(ctx, &categoryID)
	if err != nil {
		return nil, nil, err
	}
	if len(siblings) < 2 {
		siblings, err = r.fetchOrderedPublished(ctx, nil)
		if err != nil {
			return nil, nil, err
		}
	}

	idx := -1
	for i, s := range siblings {
		if s.id == currentID {
			idx = i
			break
		}
	}
	if idx == -1 || len(siblings) < 2 {
		return nil, nil, nil
	}

	var prev, next *domain.AdjacentProduct
	if idx > 0 {
		prev = &domain.AdjacentProduct{Name: siblings[idx-1].name, Slug: siblings[idx-1].slug}
	}
	if idx < len(siblings)-1 {
		next = &domain.AdjacentProduct{Name: siblings[idx+1].name, Slug: siblings[idx+1].slug}
	}
	return prev, next, nil
}

// fetchOrderedPublished mirrors the old TS sort exactly: featured first, then
// orderIndex ascending. categoryID == nil fetches the full published catalog
// (the fallback scope).
func (r *PostgresProductRepository) fetchOrderedPublished(ctx context.Context, categoryID *string) ([]adjacentRow, error) {
	query := "SELECT p.id, p.name, p.slug FROM products p WHERE p.deleted_at IS NULL AND p.is_published = true"
	args := []any{}
	if categoryID != nil {
		query += " AND p.category_id = $1"
		args = append(args, *categoryID)
	}
	query += " ORDER BY p.is_featured DESC, p.order_index ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("product repository fetchOrderedPublished: %w", err)
	}
	defer rows.Close()

	var result []adjacentRow
	for rows.Next() {
		var row adjacentRow
		if err := rows.Scan(&row.id, &row.name, &row.slug); err != nil {
			return nil, fmt.Errorf("product repository fetchOrderedPublished scan: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("product repository fetchOrderedPublished rows: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

// marshalSpecs/unmarshalSpecs hand-roll the jsonb <-> []domain.SpecItem
// conversion for the same reason marshalFAQ/unmarshalFAQ do in brand's
// postgres_repository.go (see the comment there, replicated in spirit here):
// simple protocol mode can't infer an OID for an arbitrary struct slice, and
// the return type must be the named json.RawMessage type specifically (not a
// plain []byte) or pgx encodes it as a bytea literal instead of raw JSON
// text. Unlike brand's faq column, products.specs is NOT NULL DEFAULT '{}'
// jsonb but semantically holds a JSON ARRAY — so a nil/empty Go slice must
// marshal to "[]", never to SQL NULL or "{}"; see orEmptyJSON below for the
// (unrelated) description column, which really is object-shaped.
func marshalSpecs(specs []domain.SpecItem) (json.RawMessage, error) {
	if specs == nil {
		return json.RawMessage("[]"), nil
	}
	return json.Marshal(specs)
}

func unmarshalSpecs(raw []byte) ([]domain.SpecItem, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var specs []domain.SpecItem
	if err := json.Unmarshal(raw, &specs); err != nil {
		return nil, err
	}
	return specs, nil
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
// for images/labels/normalized_specs rather than writing an explicit SQL
// NULL for "no items".
func orEmptyStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func scanProduct(row rowScanner) (*domain.Product, error) {
	var (
		id, categoryID, brandID, name, sku, slug string
		description                              json.RawMessage
		specsRaw                                 []byte
		normalizedSpecs                          []string
		images, labels                           []string
		originalPrice                            int64
		salePrice                                *int64
		discountPercent                          float64
		isFeatured, isPublished                  bool
		orderIndex                               int
		stockStatus, condition                   string
		metaTitle, metaDescription, mpn, gtin    *string
		createdAt, updatedAt                     time.Time
		deletedAt                                *time.Time
	)

	if err := row.Scan(
		&id, &categoryID, &brandID, &name, &sku, &slug,
		&description, &specsRaw, &normalizedSpecs,
		&images, &labels,
		&originalPrice, &salePrice, &discountPercent,
		&isFeatured, &isPublished, &orderIndex,
		&stockStatus, &condition,
		&metaTitle, &metaDescription, &mpn, &gtin,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	specs, err := unmarshalSpecs(specsRaw)
	if err != nil {
		return nil, fmt.Errorf("scan product (unmarshal specs): %w", err)
	}

	return domain.RehydrateProduct(
		id, categoryID, brandID, name, sku, slug,
		description, specs, normalizedSpecs,
		images, labels,
		originalPrice, salePrice, discountPercent,
		isFeatured, isPublished, orderIndex,
		stockStatus, condition,
		metaTitle, metaDescription, mpn, gtin,
		createdAt, updatedAt, deletedAt,
	), nil
}

// scanProductWithRelationsRow scans the productColumns + catalogJoin shape.
// extraDest lets callers append additional trailing SELECT columns (GetAll's
// search_rank) to the same Scan call without duplicating the whole field list.
func scanProductWithRelationsRow(row rowScanner, extraDest ...any) (*domain.ProductWithRelations, error) {
	var (
		id, categoryID, brandID, name, sku, slug string
		description                              json.RawMessage
		specsRaw                                 []byte
		normalizedSpecs                          []string
		images, labels                           []string
		originalPrice                            int64
		salePrice                                *int64
		discountPercent                          float64
		isFeatured, isPublished                  bool
		orderIndex                               int
		stockStatus, condition                   string
		metaTitle, metaDescription, mpn, gtin    *string
		createdAt, updatedAt                     time.Time
		deletedAt                                *time.Time

		catID, catName, catSlug, catMetaTitle, catMetaDescription *string

		brID, brName, brSlug, brLogoURL, brMetaTitle, brMetaDescription *string
		brIsFeatured                                                    *bool
		brOrderIndex                                                    *int
	)

	dest := []any{
		&id, &categoryID, &brandID, &name, &sku, &slug,
		&description, &specsRaw, &normalizedSpecs,
		&images, &labels,
		&originalPrice, &salePrice, &discountPercent,
		&isFeatured, &isPublished, &orderIndex,
		&stockStatus, &condition,
		&metaTitle, &metaDescription, &mpn, &gtin,
		&createdAt, &updatedAt, &deletedAt,
		&catID, &catName, &catSlug, &catMetaTitle, &catMetaDescription,
		&brID, &brName, &brSlug, &brLogoURL, &brMetaTitle, &brMetaDescription, &brIsFeatured, &brOrderIndex,
	}
	dest = append(dest, extraDest...)

	if err := row.Scan(dest...); err != nil {
		return nil, err
	}

	specs, err := unmarshalSpecs(specsRaw)
	if err != nil {
		return nil, fmt.Errorf("scan product with relations (unmarshal specs): %w", err)
	}

	product := domain.RehydrateProduct(
		id, categoryID, brandID, name, sku, slug,
		description, specs, normalizedSpecs,
		images, labels,
		originalPrice, salePrice, discountPercent,
		isFeatured, isPublished, orderIndex,
		stockStatus, condition,
		metaTitle, metaDescription, mpn, gtin,
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
