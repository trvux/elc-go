package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/project/domain"
)

type PostgresProjectRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ProjectRepository = (*PostgresProjectRepository)(nil)

func NewPostgresProjectRepository(pool *pgxpool.Pool) *PostgresProjectRepository {
	return &PostgresProjectRepository{pool: pool}
}

// projectColumns are always selected against a `p`-aliased `projects` table.
// is_published/order_index/is_featured are nullable columns in the real
// schema (no NOT NULL, only a DEFAULT) — COALESCE here reproduces the old
// TS mapper's `row.is_published || false` / `row.order_index || 0` /
// `row.is_featured || false` fallbacks exactly (see
// modules/project/infrastructure/projectRepo.ts's mapToDomain).
const projectColumns = `p.id, p.title, p.description, p.images,
	COALESCE(p.is_published, false), COALESCE(p.order_index, 0), p.created_at, p.slug, p.updated_at,
	COALESCE(p.is_featured, false), p.deleted_at, p.meta_title, p.meta_description, p.seo, p.project_type_id,
	p.client_name, p.location, p.completed_at, p.testimonial_quote, p.testimonial_author`

// marshalSeo/unmarshalSeo hand-roll the jsonb <-> domain.Seo conversion —
// the shared pool runs pgx.QueryExecModeSimpleProtocol (PgBouncer fix), which
// can't infer an OID for an arbitrary struct, same reasoning as catalog's
// marshalSpecs/unmarshalSpecs. seo is NOT NULL DEFAULT '{}' object-shaped, so
// an empty/zero Seo marshals to "{}".
func marshalSeo(seo domain.Seo) (json.RawMessage, error) {
	return json.Marshal(seo)
}

func unmarshalSeo(raw []byte) (domain.Seo, error) {
	var seo domain.Seo
	if len(raw) == 0 {
		return seo, nil
	}
	if err := json.Unmarshal(raw, &seo); err != nil {
		return domain.Seo{}, err
	}
	return seo, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func scanProject(row rowScanner) (*domain.Project, error) {
	var (
		id, title, slug                     string
		description                         json.RawMessage
		imagesRaw                           []byte
		isPublished, isFeatured             bool
		orderIndex                          int
		createdAt, updatedAt                time.Time
		deletedAt                           *time.Time
		metaTitle, metaDescription          *string
		seoRaw                              []byte
		projectTypeID                       *string
		clientName, location                string
		completedAt                         *time.Time
		testimonialQuote, testimonialAuthor string
	)

	if err := row.Scan(
		&id, &title, &description, &imagesRaw,
		&isPublished, &orderIndex, &createdAt, &slug, &updatedAt,
		&isFeatured, &deletedAt, &metaTitle, &metaDescription, &seoRaw, &projectTypeID,
		&clientName, &location, &completedAt, &testimonialQuote, &testimonialAuthor,
	); err != nil {
		return nil, err
	}

	seo, err := unmarshalSeo(seoRaw)
	if err != nil {
		return nil, fmt.Errorf("scan project (unmarshal seo): %w", err)
	}
	images, err := media.UnmarshalImages(imagesRaw)
	if err != nil {
		return nil, fmt.Errorf("scan project (unmarshal images): %w", err)
	}

	return domain.RehydrateProject(
		id, title, slug, description, images,
		isFeatured, isPublished, metaTitle, metaDescription, seo,
		orderIndex, projectTypeID,
		clientName, location, completedAt, testimonialQuote, testimonialAuthor,
		createdAt, updatedAt, deletedAt,
	), nil
}

// scanProjectWithType scans projectColumns plus a LEFT JOINed project_type's
// id/name/slug (nil pt* means the project has no project_type_id, or its
// project_type row is gone — project_type_id's FK is ON DELETE SET NULL, see
// docs/project.md).
func scanProjectWithType(row rowScanner) (*domain.Project, *domain.ProjectTypeRef, error) {
	var (
		id, title, slug                     string
		description                         json.RawMessage
		imagesRaw                           []byte
		isPublished, isFeatured             bool
		orderIndex                          int
		createdAt, updatedAt                time.Time
		deletedAt                           *time.Time
		metaTitle, metaDescription          *string
		seoRaw                              []byte
		projectTypeID                       *string
		clientName, location                string
		completedAt                         *time.Time
		testimonialQuote, testimonialAuthor string
		ptID, ptName, ptSlug                *string
	)

	if err := row.Scan(
		&id, &title, &description, &imagesRaw,
		&isPublished, &orderIndex, &createdAt, &slug, &updatedAt,
		&isFeatured, &deletedAt, &metaTitle, &metaDescription, &seoRaw, &projectTypeID,
		&clientName, &location, &completedAt, &testimonialQuote, &testimonialAuthor,
		&ptID, &ptName, &ptSlug,
	); err != nil {
		return nil, nil, err
	}

	seo, err := unmarshalSeo(seoRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("scan project with type (unmarshal seo): %w", err)
	}
	images, err := media.UnmarshalImages(imagesRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("scan project with type (unmarshal images): %w", err)
	}

	project := domain.RehydrateProject(
		id, title, slug, description, images,
		isFeatured, isPublished, metaTitle, metaDescription, seo,
		orderIndex, projectTypeID,
		clientName, location, completedAt, testimonialQuote, testimonialAuthor,
		createdAt, updatedAt, deletedAt,
	)

	var pt *domain.ProjectTypeRef
	if ptID != nil {
		pt = &domain.ProjectTypeRef{ID: *ptID, Name: derefStr(ptName), Slug: derefStr(ptSlug)}
	}

	return project, pt, nil
}

// buildProjectConditions returns the WHERE conditions + positional args
// shared by GetAll and Count. categorySlug(s)/serviceSlug(s) are resolved
// via EXISTS subqueries in one round trip — the old TS repository resolved
// each of these with a separate round-trip query per filter before then
// filtering by resolved IDs (see modules/project/infrastructure/
// projectRepo.ts's getAll), which is exactly the kind of avoidable extra
// round trip ARCHITECTURE.md's Performance Checklist (§15) calls out.
func buildProjectConditions(filter domain.ProjectFilter) ([]string, []any) {
	var conditions []string
	var args []any
	argN := 1
	next := func() int {
		n := argN
		argN++
		return n
	}

	if !filter.IncludeDeleted {
		conditions = append(conditions, "p.deleted_at IS NULL")
	}
	if filter.ProjectTypeID != nil {
		conditions = append(conditions, fmt.Sprintf("p.project_type_id = $%d", next()))
		args = append(args, *filter.ProjectTypeID)
	}
	if filter.ExcludeID != nil {
		conditions = append(conditions, fmt.Sprintf("p.id != $%d", next()))
		args = append(args, *filter.ExcludeID)
	}
	if filter.CategorySlug != nil {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM project_category pc JOIN categories c ON c.id = pc.category_id WHERE pc.project_id = p.id AND c.slug = $%d AND c.deleted_at IS NULL)",
			next(),
		))
		args = append(args, *filter.CategorySlug)
	}
	if len(filter.CategorySlugs) > 0 {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM project_category pc JOIN categories c ON c.id = pc.category_id WHERE pc.project_id = p.id AND c.slug = ANY($%d) AND c.deleted_at IS NULL)",
			next(),
		))
		args = append(args, filter.CategorySlugs)
	}
	if filter.ServiceSlug != nil {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM project_service ps JOIN services s ON s.id = ps.service_id WHERE ps.project_id = p.id AND s.slug = $%d AND s.deleted_at IS NULL)",
			next(),
		))
		args = append(args, *filter.ServiceSlug)
	}
	if len(filter.ServiceSlugs) > 0 {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM project_service ps JOIN services s ON s.id = ps.service_id WHERE ps.project_id = p.id AND s.slug = ANY($%d) AND s.deleted_at IS NULL)",
			next(),
		))
		args = append(args, filter.ServiceSlugs)
	}
	if filter.IsPublished != nil {
		conditions = append(conditions, fmt.Sprintf("p.is_published = $%d", next()))
		args = append(args, *filter.IsPublished)
	}
	if filter.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("p.is_featured = $%d", next()))
		args = append(args, *filter.IsFeatured)
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("p.title ILIKE $%d", next()))
		args = append(args, "%"+filter.Search+"%")
	}

	return conditions, args
}

func orderByClause(filter domain.ProjectFilter) string {
	col := "p.order_index"
	switch filter.OrderBy {
	case "createdAt":
		col = "p.created_at"
	case "title":
		col = "p.title"
	}
	dir := "ASC"
	if filter.OrderDirection == "desc" {
		dir = "DESC"
	}
	return col + " " + dir
}

func (r *PostgresProjectRepository) GetAll(ctx context.Context, filter domain.ProjectFilter) ([]*domain.ProjectWithRelations, error) {
	conditions, args := buildProjectConditions(filter)

	query := "SELECT " + projectColumns + `, pt.id, pt.name, pt.slug
		FROM projects p
		LEFT JOIN project_type pt ON pt.id = p.project_type_id`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY " + orderByClause(filter)

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("project repository getAll: %w", err)
	}
	defer rows.Close()

	var projects []*domain.Project
	projectTypes := map[string]*domain.ProjectTypeRef{}
	var ids []string
	for rows.Next() {
		p, pt, err := scanProjectWithType(rows)
		if err != nil {
			return nil, fmt.Errorf("project repository getAll scan: %w", err)
		}
		projects = append(projects, p)
		ids = append(ids, p.ID())
		if pt != nil {
			projectTypes[p.ID()] = pt
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("project repository getAll rows: %w", err)
	}

	categoriesByProject, err := r.fetchCategoriesForProjects(ctx, ids, false)
	if err != nil {
		return nil, err
	}
	servicesByProject, err := r.fetchServicesForProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	tagsByProject, err := fetchTagsForProjects(ctx, r.pool, ids)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ProjectWithRelations, 0, len(projects))
	for _, p := range projects {
		result = append(result, &domain.ProjectWithRelations{
			Project:     p,
			ProjectType: projectTypes[p.ID()],
			Categories:  categoriesByProject[p.ID()],
			Services:    servicesByProject[p.ID()],
			Tags:        tagsByProject[p.ID()],
		})
	}
	return result, nil
}

func (r *PostgresProjectRepository) Count(ctx context.Context, filter domain.ProjectFilter) (int, error) {
	conditions, args := buildProjectConditions(filter)
	query := "SELECT COUNT(*) FROM projects p"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("project repository count: %w", err)
	}
	return count, nil
}

func (r *PostgresProjectRepository) GetByID(ctx context.Context, id string) (*domain.ProjectWithRelations, error) {
	query := "SELECT " + projectColumns + `, pt.id, pt.name, pt.slug
		FROM projects p
		LEFT JOIN project_type pt ON pt.id = p.project_type_id
		WHERE p.id = $1 AND p.deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, query, id)
	p, pt, err := scanProjectWithType(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("project repository getById: %w", err)
	}

	return r.attachRelations(ctx, p, pt, false)
}

func (r *PostgresProjectRepository) GetBySlug(ctx context.Context, slug string, withPricing bool) (*domain.ProjectWithRelations, error) {
	query := "SELECT " + projectColumns + `, pt.id, pt.name, pt.slug
		FROM projects p
		LEFT JOIN project_type pt ON pt.id = p.project_type_id
		WHERE p.slug = $1 AND p.deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, query, slug)
	p, pt, err := scanProjectWithType(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("project repository getBySlug: %w", err)
	}

	return r.attachRelations(ctx, p, pt, withPricing)
}

func (r *PostgresProjectRepository) attachRelations(ctx context.Context, p *domain.Project, pt *domain.ProjectTypeRef, withPricing bool) (*domain.ProjectWithRelations, error) {
	categoriesByProject, err := r.fetchCategoriesForProjects(ctx, []string{p.ID()}, withPricing)
	if err != nil {
		return nil, err
	}
	servicesByProject, err := r.fetchServicesForProjects(ctx, []string{p.ID()})
	if err != nil {
		return nil, err
	}
	tagsByProject, err := fetchTagsForProjects(ctx, r.pool, []string{p.ID()})
	if err != nil {
		return nil, err
	}
	return &domain.ProjectWithRelations{
		Project:     p,
		ProjectType: pt,
		Categories:  categoriesByProject[p.ID()],
		Services:    servicesByProject[p.ID()],
		Tags:        tagsByProject[p.ID()],
	}, nil
}

// pgxQuerier is satisfied by both *pgxpool.Pool and pgx.Tx.
type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// fetchTagsForProjects batch-loads project_tags rows (+ tag name/slug) for
// every project id in one round trip, keyed by project id — same pattern as
// fetchCategoriesForProjects/fetchServicesForProjects above.
func fetchTagsForProjects(ctx context.Context, q pgxQuerier, projectIDs []string) (map[string][]domain.TagRef, error) {
	if len(projectIDs) == 0 {
		return map[string][]domain.TagRef{}, nil
	}

	query := `
		SELECT pt.project_id, t.id, t.name, t.slug
		FROM project_tags pt
		JOIN tags t ON t.id = pt.tag_id AND t.deleted_at IS NULL
		WHERE pt.project_id = ANY($1)
		ORDER BY t.name ASC`

	rows, err := q.Query(ctx, query, projectIDs)
	if err != nil {
		return nil, fmt.Errorf("project repository fetchTags: %w", err)
	}
	defer rows.Close()

	result := map[string][]domain.TagRef{}
	for rows.Next() {
		var projectID string
		var tag domain.TagRef
		if err := rows.Scan(&projectID, &tag.ID, &tag.Name, &tag.Slug); err != nil {
			return nil, fmt.Errorf("project repository fetchTags scan: %w", err)
		}
		result[projectID] = append(result[projectID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("project repository fetchTags rows: %w", err)
	}
	return result, nil
}

func insertProjectTags(ctx context.Context, tx pgx.Tx, projectID string, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO project_tags (project_id, tag_id) VALUES ")
	args := make([]any, 0, len(tagIDs)*2)
	argN := 1
	for i, tagID := range tagIDs {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("($%d, $%d)", argN, argN+1))
		args = append(args, projectID, tagID)
		argN += 2
	}
	if _, err := tx.Exec(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("project repository insertTags: %w", err)
	}
	return nil
}

// fetchCategoriesForProjects batch-loads project_category rows (+ category +
// its group) for every project id in one round trip, keyed by project id —
// avoids the N+1 the naive per-project version would cause. withPricing
// turns on the LATERAL join into `products` that computes lowPrice/
// highPrice/offerCount — mirrors the old TS split between
// projectRepo.ts's getAll/getById/getBySlug (no pricing) and
// infrastructure/resolveProjectPath.ts (pricing), see docs/project.md.
func (r *PostgresProjectRepository) fetchCategoriesForProjects(ctx context.Context, projectIDs []string, withPricing bool) (map[string][]domain.ProjectCategory, error) {
	if len(projectIDs) == 0 {
		return map[string][]domain.ProjectCategory{}, nil
	}

	priceSelect := "0::bigint, 0::bigint, 0::int"
	priceJoin := ""
	if withPricing {
		// Mirrors the old TS `p.sale_price || p.original_price || 0` (JS
		// treats 0 as falsy too) via COALESCE+NULLIF, then only counts/
		// ranges over strictly-positive prices — same as the old
		// `.filter((p) => p > 0)` in resolveProjectPath.ts.
		priceSelect = "COALESCE(price_agg.low_price, 0), COALESCE(price_agg.high_price, 0), COALESCE(price_agg.offer_count, 0)"
		priceJoin = `
		LEFT JOIN LATERAL (
			SELECT MIN(price) AS low_price, MAX(price) AS high_price, COUNT(*) AS offer_count
			FROM (
				SELECT COALESCE(NULLIF(pr.sale_price, 0), NULLIF(pr.original_price, 0), 0) AS price
				FROM products pr
				WHERE pr.category_id = c.id AND pr.is_published = true AND pr.deleted_at IS NULL
			) prices
			WHERE price > 0
		) price_agg ON true`
	}

	query := `
		SELECT pc.project_id, pc.condition, c.id, c.name, c.slug, c.group_id, gc.id, gc.name, ` + priceSelect + `
		FROM project_category pc
		JOIN categories c ON c.id = pc.category_id
		LEFT JOIN group_categories gc ON gc.id = c.group_id` + priceJoin + `
		WHERE pc.project_id = ANY($1)`

	rows, err := r.pool.Query(ctx, query, projectIDs)
	if err != nil {
		return nil, fmt.Errorf("project repository fetchCategories: %w", err)
	}
	defer rows.Close()

	result := map[string][]domain.ProjectCategory{}
	for rows.Next() {
		var (
			projectID, condition, catID, catName string
			catSlug                              *string
			groupID                              *string
			groupCatID, groupCatName             *string
			lowPrice, highPrice                  int64
			offerCount                           int
		)
		if err := rows.Scan(
			&projectID, &condition, &catID, &catName, &catSlug, &groupID, &groupCatID, &groupCatName,
			&lowPrice, &highPrice, &offerCount,
		); err != nil {
			return nil, fmt.Errorf("project repository fetchCategories scan: %w", err)
		}

		var group *domain.CategoryGroupRef
		if groupCatID != nil {
			group = &domain.CategoryGroupRef{ID: *groupCatID, Name: derefStr(groupCatName)}
		}

		result[projectID] = append(result[projectID], domain.ProjectCategory{
			ID: catID, Name: catName, Slug: derefStr(catSlug), GroupID: groupID, Condition: condition,
			Group: group, LowPrice: lowPrice, HighPrice: highPrice, OfferCount: offerCount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("project repository fetchCategories rows: %w", err)
	}
	return result, nil
}

// fetchServicesForProjects batch-loads project_service rows (+ service +
// its group) for every project id in one round trip, same reasoning as
// fetchCategoriesForProjects.
func (r *PostgresProjectRepository) fetchServicesForProjects(ctx context.Context, projectIDs []string) (map[string][]domain.ProjectServiceRef, error) {
	if len(projectIDs) == 0 {
		return map[string][]domain.ProjectServiceRef{}, nil
	}

	query := `
		SELECT ps.project_id, s.id, s.title, s.slug, sg.id, sg.name, sg.slug
		FROM project_service ps
		JOIN services s ON s.id = ps.service_id
		LEFT JOIN service_groups sg ON sg.id = s.group_id
		WHERE ps.project_id = ANY($1)`

	rows, err := r.pool.Query(ctx, query, projectIDs)
	if err != nil {
		return nil, fmt.Errorf("project repository fetchServices: %w", err)
	}
	defer rows.Close()

	result := map[string][]domain.ProjectServiceRef{}
	for rows.Next() {
		var (
			projectID, svcID, svcTitle    string
			svcSlug                       *string
			groupID, groupName, groupSlug *string
		)
		if err := rows.Scan(&projectID, &svcID, &svcTitle, &svcSlug, &groupID, &groupName, &groupSlug); err != nil {
			return nil, fmt.Errorf("project repository fetchServices scan: %w", err)
		}

		var group *domain.ServiceGroupRef
		if groupID != nil {
			group = &domain.ServiceGroupRef{ID: *groupID, Name: derefStr(groupName), Slug: derefStr(groupSlug)}
		}

		result[projectID] = append(result[projectID], domain.ProjectServiceRef{
			ID: svcID, Title: svcTitle, Slug: derefStr(svcSlug), Group: group,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("project repository fetchServices rows: %w", err)
	}
	return result, nil
}

// Create resurrects a soft-deleted row sharing the same slug rather than
// plain-INSERTing — projects.slug is a plain UNIQUE constraint (unlike
// brand/group/category's partial "unique among non-deleted" index), so a
// plain INSERT would fail the constraint outright on slug reuse. Both the
// resurrect-or-insert step and the relation writes run in one transaction:
// the old TS create() ran the equivalent as several sequential,
// un-transactioned Supabase calls (find-existing, insert/update, delete old
// relations, insert new relations) — a real partial-failure risk this fixes,
// same spirit as service-group/group's transactional SoftDelete. See
// docs/project.md.
func (r *PostgresProjectRepository) Create(ctx context.Context, project *domain.Project, categories []domain.CategoryCondition, serviceIDs []string, tagIDs []string) (*domain.Project, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("project repository create (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	var existingID string
	err = tx.QueryRow(ctx,
		"SELECT id FROM projects WHERE slug = $1 AND deleted_at IS NOT NULL",
		project.Slug(),
	).Scan(&existingID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("project repository create (check existing): %w", err)
	}
	isResurrect := err == nil

	seoJSON, err := marshalSeo(project.Seo())
	if err != nil {
		return nil, fmt.Errorf("project repository create (marshal seo): %w", err)
	}
	imagesJSON, err := media.MarshalImages(project.Images())
	if err != nil {
		return nil, fmt.Errorf("project repository create (marshal images): %w", err)
	}

	var createdID string
	if isResurrect {
		query := `
			UPDATE projects
			SET title = $1, description = $2, images = $3,
				is_published = $4, order_index = $5, slug = $6,
				is_featured = $7, meta_title = $8, meta_description = $9, seo = $10,
				project_type_id = $11, client_name = $12, location = $13, completed_at = $14,
				testimonial_quote = $15, testimonial_author = $16, deleted_at = NULL, updated_at = $17
			WHERE id = $18
			RETURNING id`
		if err := tx.QueryRow(ctx, query,
			project.Title(), project.Description(), imagesJSON,
			project.IsPublished(), project.OrderIndex(), project.Slug(),
			project.IsFeatured(), project.MetaTitle(), project.MetaDescription(), seoJSON,
			project.ProjectTypeID(), project.ClientName(), project.Location(), project.CompletedAt(),
			project.TestimonialQuote(), project.TestimonialAuthor(), time.Now(), existingID,
		).Scan(&createdID); err != nil {
			return nil, fmt.Errorf("project repository create (resurrect): %w", err)
		}

		// Old TS create()'s resurrect path always starts a resurrected
		// project's relations fresh rather than merging with whatever the
		// soft-deleted row used to have.
		if _, err := tx.Exec(ctx, "DELETE FROM project_category WHERE project_id = $1", createdID); err != nil {
			return nil, fmt.Errorf("project repository create (clear categories): %w", err)
		}
		if _, err := tx.Exec(ctx, "DELETE FROM project_service WHERE project_id = $1", createdID); err != nil {
			return nil, fmt.Errorf("project repository create (clear services): %w", err)
		}
		if _, err := tx.Exec(ctx, "DELETE FROM project_tags WHERE project_id = $1", createdID); err != nil {
			return nil, fmt.Errorf("project repository create (clear tags): %w", err)
		}
	} else {
		query := `
			INSERT INTO projects (title, description, images, is_published, order_index, slug, is_featured, meta_title, meta_description, seo, project_type_id, client_name, location, completed_at, testimonial_quote, testimonial_author)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			RETURNING id`
		if err := tx.QueryRow(ctx, query,
			project.Title(), project.Description(), imagesJSON,
			project.IsPublished(), project.OrderIndex(), project.Slug(),
			project.IsFeatured(), project.MetaTitle(), project.MetaDescription(), seoJSON,
			project.ProjectTypeID(), project.ClientName(), project.Location(), project.CompletedAt(),
			project.TestimonialQuote(), project.TestimonialAuthor(),
		).Scan(&createdID); err != nil {
			return nil, fmt.Errorf("project repository create: %w", err)
		}
	}

	if err := insertProjectCategories(ctx, tx, createdID, categories); err != nil {
		return nil, err
	}
	if err := insertProjectServices(ctx, tx, createdID, serviceIDs); err != nil {
		return nil, err
	}
	if err := insertProjectTags(ctx, tx, createdID, tagIDs); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, "SELECT "+projectColumns+" FROM projects p WHERE p.id = $1", createdID)
	created, err := scanProject(row)
	if err != nil {
		return nil, fmt.Errorf("project repository create (reload): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("project repository create (commit tx): %w", err)
	}
	return created, nil
}

// Update runs the row update + full relation replace in one transaction,
// same atomicity fix as Create — see its doc comment.
func (r *PostgresProjectRepository) Update(ctx context.Context, project *domain.Project, categories *[]domain.CategoryCondition, serviceIDs *[]string, tagIDs *[]string) (*domain.Project, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("project repository update (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	seoJSON, err := marshalSeo(project.Seo())
	if err != nil {
		return nil, fmt.Errorf("project repository update (marshal seo): %w", err)
	}
	imagesJSON, err := media.MarshalImages(project.Images())
	if err != nil {
		return nil, fmt.Errorf("project repository update (marshal images): %w", err)
	}

	query := `
		UPDATE projects p
		SET title = $1, description = $2, images = $3,
			is_published = $4, order_index = $5, slug = $6,
			is_featured = $7, meta_title = $8, meta_description = $9, seo = $10,
			project_type_id = $11, client_name = $12, location = $13, completed_at = $14,
			testimonial_quote = $15, testimonial_author = $16, updated_at = $17
		WHERE p.id = $18
		RETURNING ` + projectColumns

	row := tx.QueryRow(ctx, query,
		project.Title(), project.Description(), imagesJSON,
		project.IsPublished(), project.OrderIndex(), project.Slug(),
		project.IsFeatured(), project.MetaTitle(), project.MetaDescription(), seoJSON,
		project.ProjectTypeID(), project.ClientName(), project.Location(), project.CompletedAt(),
		project.TestimonialQuote(), project.TestimonialAuthor(), project.UpdatedAt(), project.ID(),
	)
	updated, err := scanProject(row)
	if err != nil {
		return nil, fmt.Errorf("project repository update: %w", err)
	}

	if categories != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM project_category WHERE project_id = $1", project.ID()); err != nil {
			return nil, fmt.Errorf("project repository update (clear categories): %w", err)
		}
		if err := insertProjectCategories(ctx, tx, project.ID(), *categories); err != nil {
			return nil, err
		}
	}
	if serviceIDs != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM project_service WHERE project_id = $1", project.ID()); err != nil {
			return nil, fmt.Errorf("project repository update (clear services): %w", err)
		}
		if err := insertProjectServices(ctx, tx, project.ID(), *serviceIDs); err != nil {
			return nil, err
		}
	}
	if tagIDs != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM project_tags WHERE project_id = $1", project.ID()); err != nil {
			return nil, fmt.Errorf("project repository update (clear tags): %w", err)
		}
		if err := insertProjectTags(ctx, tx, project.ID(), *tagIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("project repository update (commit tx): %w", err)
	}
	return updated, nil
}

func insertProjectCategories(ctx context.Context, tx pgx.Tx, projectID string, categories []domain.CategoryCondition) error {
	if len(categories) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO project_category (project_id, category_id, condition) VALUES ")
	args := make([]any, 0, len(categories)*3)
	argN := 1
	for i, c := range categories {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d)", argN, argN+1, argN+2))
		args = append(args, projectID, c.CategoryID, c.Condition)
		argN += 3
	}
	if _, err := tx.Exec(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("project repository insertCategories: %w", err)
	}
	return nil
}

func insertProjectServices(ctx context.Context, tx pgx.Tx, projectID string, serviceIDs []string) error {
	if len(serviceIDs) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO project_service (project_id, service_id) VALUES ")
	args := make([]any, 0, len(serviceIDs)*2)
	argN := 1
	for i, sid := range serviceIDs {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("($%d, $%d)", argN, argN+1))
		args = append(args, projectID, sid)
		argN += 2
	}
	if _, err := tx.Exec(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("project repository insertServices: %w", err)
	}
	return nil
}

// SoftDelete only ever touches the projects row itself — the old TS
// delete() never cleaned up project_category/project_service either,
// leaving those join rows pointing at a soft-deleted project (their FK is
// ON DELETE CASCADE, but soft-delete is an UPDATE, so cascade never fires;
// same "leftover FK reference" precedent as brand's products.brand_id, see
// docs/brand.md). See docs/project.md.
func (r *PostgresProjectRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE projects SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("project repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresProjectRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE projects SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("project repository restore: %w", err)
	}
	return nil
}

// UpdateOrder/TogglePublish/ToggleFeatured deliberately don't set updated_at
// themselves — same as the old TS repository's single-column updates —
// because the `update_projects_modtime` trigger already sets it on every
// UPDATE regardless of what the client sends. See docs/project.md.
func (r *PostgresProjectRepository) UpdateOrder(ctx context.Context, id string, orderIndex int) error {
	_, err := r.pool.Exec(ctx, "UPDATE projects SET order_index = $1 WHERE id = $2", orderIndex, id)
	if err != nil {
		return fmt.Errorf("project repository updateOrder: %w", err)
	}
	return nil
}

func (r *PostgresProjectRepository) TogglePublish(ctx context.Context, id string, isPublished bool) error {
	_, err := r.pool.Exec(ctx, "UPDATE projects SET is_published = $1 WHERE id = $2", isPublished, id)
	if err != nil {
		return fmt.Errorf("project repository togglePublish: %w", err)
	}
	return nil
}

func (r *PostgresProjectRepository) ToggleFeatured(ctx context.Context, id string, isFeatured bool) error {
	_, err := r.pool.Exec(ctx, "UPDATE projects SET is_featured = $1 WHERE id = $2", isFeatured, id)
	if err != nil {
		return fmt.Errorf("project repository toggleFeatured: %w", err)
	}
	return nil
}

type adjacentProjectRow struct {
	id, title, slug string
}

// GetAdjacent ports internal/catalog's GetAdjacent pattern (same "same-group
// siblings first, fall back to full published set" rule) — the old TS
// getAdjacentProjects.ts loaded every published project into memory to sort;
// this pushes that down to two indexed SQL queries at most (the fallback
// only runs when the first came back too small). See domain/repository.go.
func (r *PostgresProjectRepository) GetAdjacent(ctx context.Context, projectTypeID *string, currentID string) (*domain.AdjacentProject, *domain.AdjacentProject, error) {
	siblings, err := r.fetchOrderedPublished(ctx, projectTypeID)
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

	var prev, next *domain.AdjacentProject
	if idx > 0 {
		prev = &domain.AdjacentProject{Title: siblings[idx-1].title, Slug: siblings[idx-1].slug}
	}
	if idx < len(siblings)-1 {
		next = &domain.AdjacentProject{Title: siblings[idx+1].title, Slug: siblings[idx+1].slug}
	}
	return prev, next, nil
}

// fetchOrderedPublished mirrors the old TS sort exactly: featured first,
// then orderIndex ascending. projectTypeID == nil fetches the full published
// set (the fallback scope).
func (r *PostgresProjectRepository) fetchOrderedPublished(ctx context.Context, projectTypeID *string) ([]adjacentProjectRow, error) {
	query := "SELECT id, title, slug FROM projects WHERE deleted_at IS NULL AND is_published = true"
	args := []any{}
	if projectTypeID != nil {
		query += " AND project_type_id = $1"
		args = append(args, *projectTypeID)
	}
	query += " ORDER BY is_featured DESC, order_index ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("project repository fetchOrderedPublished: %w", err)
	}
	defer rows.Close()

	var result []adjacentProjectRow
	for rows.Next() {
		var row adjacentProjectRow
		if err := rows.Scan(&row.id, &row.title, &row.slug); err != nil {
			return nil, fmt.Errorf("project repository fetchOrderedPublished scan: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("project repository fetchOrderedPublished rows: %w", err)
	}
	return result, nil
}

// GetCategoriesByProjectTypeID ports the old TS getCategoriesByProjectTypeId
// (modules/project/infrastructure/projectRepo.ts) as a single query with
// DISTINCT instead of fetching+deduping in application code.
func (r *PostgresProjectRepository) GetCategoriesByProjectTypeID(ctx context.Context, projectTypeID string) ([]domain.CategoryRef, error) {
	query := `
		SELECT DISTINCT c.id, c.name, c.slug
		FROM project_category pc
		JOIN projects p ON p.id = pc.project_id
		JOIN categories c ON c.id = pc.category_id
		WHERE p.project_type_id = $1 AND p.is_published = true AND p.deleted_at IS NULL AND c.deleted_at IS NULL
		ORDER BY c.name ASC`

	rows, err := r.pool.Query(ctx, query, projectTypeID)
	if err != nil {
		return nil, fmt.Errorf("project repository getCategoriesByProjectTypeId: %w", err)
	}
	defer rows.Close()

	var result []domain.CategoryRef
	for rows.Next() {
		var c domain.CategoryRef
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug); err != nil {
			return nil, fmt.Errorf("project repository getCategoriesByProjectTypeId scan: %w", err)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("project repository getCategoriesByProjectTypeId rows: %w", err)
	}
	return result, nil
}
