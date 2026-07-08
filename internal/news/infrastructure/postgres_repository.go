package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/news/domain"
	"github.com/trvux/elc-go/internal/platform/media"
)

type PostgresNewsRepository struct {
	pool *pgxpool.Pool
}

var _ domain.NewsRepository = (*PostgresNewsRepository)(nil)

func NewPostgresNewsRepository(pool *pgxpool.Pool) *PostgresNewsRepository {
	return &PostgresNewsRepository{pool: pool}
}

const newsColumns = "id, title, slug, images, content, excerpt, category_id, author_id, is_published, meta_title, meta_description, seo, order_index, created_at, updated_at, deleted_at"

type rowScanner interface {
	Scan(dest ...any) error
}

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

func scanNews(row rowScanner) (*domain.News, error) {
	var (
		id, title, slug            string
		imagesRaw                  []byte
		content                    json.RawMessage
		excerpt                    string
		categoryID                 *string
		authorID                   *string
		isPublished                bool
		metaTitle, metaDescription *string
		seoRaw                     []byte
		orderIndex                 int
		createdAt, updatedAt       time.Time
		deletedAt                  *time.Time
	)

	if err := row.Scan(
		&id, &title, &slug, &imagesRaw, &content, &excerpt, &categoryID, &authorID,
		&isPublished, &metaTitle, &metaDescription, &seoRaw, &orderIndex,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	seo, err := unmarshalSeo(seoRaw)
	if err != nil {
		return nil, fmt.Errorf("scan news (unmarshal seo): %w", err)
	}

	images, err := media.UnmarshalImages(imagesRaw)
	if err != nil {
		return nil, fmt.Errorf("scan news (unmarshal images): %w", err)
	}

	return domain.RehydrateNews(
		id, title, slug, images, content, excerpt, categoryID, authorID,
		isPublished, metaTitle, metaDescription, seo, orderIndex,
		createdAt, updatedAt, deletedAt,
	), nil
}

// pgxQuerier is satisfied by both *pgxpool.Pool and pgx.Tx — lets
// fetchTagsForNews run either inside a transaction (Create/Update, reading
// back rows just written in the same tx) or standalone (GetAll/GetByID/
// GetBySlug).
type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// fetchTagsForNews batch-loads news_tags rows (+ tag name/slug) for every
// news id in one round trip, keyed by news id — same N+1-avoidance pattern
// as project's fetchCategoriesForProjects.
func fetchTagsForNews(ctx context.Context, q pgxQuerier, newsIDs []string) (map[string][]domain.TagRef, error) {
	if len(newsIDs) == 0 {
		return map[string][]domain.TagRef{}, nil
	}

	query := `
		SELECT nt.news_id, t.id, t.name, t.slug
		FROM news_tags nt
		JOIN tags t ON t.id = nt.tag_id AND t.deleted_at IS NULL
		WHERE nt.news_id = ANY($1)
		ORDER BY t.name ASC`

	rows, err := q.Query(ctx, query, newsIDs)
	if err != nil {
		return nil, fmt.Errorf("news repository fetchTags: %w", err)
	}
	defer rows.Close()

	result := map[string][]domain.TagRef{}
	for rows.Next() {
		var newsID string
		var tag domain.TagRef
		if err := rows.Scan(&newsID, &tag.ID, &tag.Name, &tag.Slug); err != nil {
			return nil, fmt.Errorf("news repository fetchTags scan: %w", err)
		}
		result[newsID] = append(result[newsID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("news repository fetchTags rows: %w", err)
	}
	return result, nil
}

func insertNewsTags(ctx context.Context, tx pgx.Tx, newsID string, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO news_tags (news_id, tag_id) VALUES ")
	args := make([]any, 0, len(tagIDs)*2)
	argN := 1
	for i, tagID := range tagIDs {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("($%d, $%d)", argN, argN+1))
		args = append(args, newsID, tagID)
		argN += 2
	}
	if _, err := tx.Exec(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("news repository insertTags: %w", err)
	}
	return nil
}

func buildNewsConditions(filter domain.NewsFilter) ([]string, []any) {
	var conditions []string
	var args []any
	argN := 1
	next := func() int {
		n := argN
		argN++
		return n
	}

	if !filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filter.IsPublished != nil {
		conditions = append(conditions, fmt.Sprintf("is_published = $%d", next()))
		args = append(args, *filter.IsPublished)
	}
	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", next()))
		args = append(args, *filter.CategoryID)
	}
	if filter.ExcludeID != nil {
		conditions = append(conditions, fmt.Sprintf("id != $%d", next()))
		args = append(args, *filter.ExcludeID)
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("title ILIKE $%d", next()))
		args = append(args, "%"+filter.Search+"%")
	}

	return conditions, args
}

func newsOrderByClause(filter domain.NewsFilter) string {
	col := "order_index"
	if filter.SortBy == "created_at" {
		col = "created_at"
	}
	dir := "ASC"
	if filter.SortOrder == "desc" {
		dir = "DESC"
	}
	return col + " " + dir
}

func (r *PostgresNewsRepository) GetAll(ctx context.Context, filter domain.NewsFilter) ([]*domain.News, error) {
	conditions, args := buildNewsConditions(filter)

	query := "SELECT " + newsColumns + " FROM news"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY " + newsOrderByClause(filter)

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("news repository getAll: %w", err)
	}
	defer rows.Close()

	var result []*domain.News
	for rows.Next() {
		n, err := scanNews(rows)
		if err != nil {
			return nil, fmt.Errorf("news repository getAll scan: %w", err)
		}
		result = append(result, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("news repository getAll rows: %w", err)
	}

	ids := make([]string, len(result))
	for i, n := range result {
		ids[i] = n.ID()
	}
	tagsByNews, err := fetchTagsForNews(ctx, r.pool, ids)
	if err != nil {
		return nil, err
	}
	for _, n := range result {
		n.SetTags(tagsByNews[n.ID()])
	}

	return result, nil
}

func (r *PostgresNewsRepository) Count(ctx context.Context, filter domain.NewsFilter) (int, error) {
	conditions, args := buildNewsConditions(filter)
	query := "SELECT COUNT(*) FROM news"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("news repository count: %w", err)
	}
	return count, nil
}

func (r *PostgresNewsRepository) GetByID(ctx context.Context, id string) (*domain.News, error) {
	query := "SELECT " + newsColumns + " FROM news WHERE id = $1 AND deleted_at IS NULL"
	row := r.pool.QueryRow(ctx, query, id)
	n, err := scanNews(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("news repository getById: %w", err)
	}
	if err := r.attachTags(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (r *PostgresNewsRepository) GetBySlug(ctx context.Context, slug string) (*domain.News, error) {
	query := "SELECT " + newsColumns + " FROM news WHERE slug = $1 AND deleted_at IS NULL"
	row := r.pool.QueryRow(ctx, query, slug)
	n, err := scanNews(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("news repository getBySlug: %w", err)
	}
	if err := r.attachTags(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (r *PostgresNewsRepository) attachTags(ctx context.Context, n *domain.News) error {
	tagsByNews, err := fetchTagsForNews(ctx, r.pool, []string{n.ID()})
	if err != nil {
		return err
	}
	n.SetTags(tagsByNews[n.ID()])
	return nil
}

// Create resurrects a soft-deleted row sharing the same slug rather than
// plain-INSERTing — news.slug is a plain UNIQUE constraint, so a plain
// INSERT would fail outright on slug reuse. See domain/repository.go and
// docs/news.md.
func (r *PostgresNewsRepository) Create(ctx context.Context, news *domain.News, tagIDs []string) (*domain.News, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("news repository create (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	var existingID string
	err = tx.QueryRow(ctx,
		"SELECT id FROM news WHERE slug = $1 AND deleted_at IS NOT NULL",
		news.Slug(),
	).Scan(&existingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("news repository create (check existing): %w", err)
	}
	isResurrect := err == nil

	seoJSON, err := marshalSeo(news.Seo())
	if err != nil {
		return nil, fmt.Errorf("news repository create (marshal seo): %w", err)
	}

	imagesJSON, err := media.MarshalImages(news.Images())
	if err != nil {
		return nil, fmt.Errorf("news repository create (marshal images): %w", err)
	}

	var row pgx.Row
	if isResurrect {
		query := `
			UPDATE news
			SET title = $1, slug = $2, images = $3, content = $4, excerpt = $5, category_id = $6, author_id = $7,
				is_published = $8, meta_title = $9, meta_description = $10, seo = $11, order_index = $12,
				deleted_at = NULL, updated_at = $13
			WHERE id = $14
			RETURNING ` + newsColumns
		row = tx.QueryRow(ctx, query,
			news.Title(), news.Slug(), imagesJSON, news.Content(), news.Excerpt(), news.CategoryID(), news.AuthorID(),
			news.IsPublished(), news.MetaTitle(), news.MetaDescription(), seoJSON, news.OrderIndex(),
			time.Now(), existingID,
		)
	} else {
		query := `
			INSERT INTO news (title, slug, images, content, excerpt, category_id, author_id, is_published, meta_title, meta_description, seo, order_index)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING ` + newsColumns
		row = tx.QueryRow(ctx, query,
			news.Title(), news.Slug(), imagesJSON, news.Content(), news.Excerpt(), news.CategoryID(), news.AuthorID(),
			news.IsPublished(), news.MetaTitle(), news.MetaDescription(), seoJSON, news.OrderIndex(),
		)
	}

	created, err := scanNews(row)
	if err != nil {
		return nil, fmt.Errorf("news repository create: %w", err)
	}

	// Resurrect path: old TS create() always started a resurrected row's
	// relations fresh rather than merging with whatever the soft-deleted row
	// used to have — same precedent as project's Create.
	if isResurrect {
		if _, err := tx.Exec(ctx, "DELETE FROM news_tags WHERE news_id = $1", created.ID()); err != nil {
			return nil, fmt.Errorf("news repository create (clear tags): %w", err)
		}
	}
	if err := insertNewsTags(ctx, tx, created.ID(), tagIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("news repository create (commit tx): %w", err)
	}

	created.SetTags(nil)
	if err := r.attachTags(ctx, created); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *PostgresNewsRepository) Update(ctx context.Context, news *domain.News, tagIDs *[]string) (*domain.News, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("news repository update (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE news
		SET title = $1, slug = $2, images = $3, content = $4, excerpt = $5, category_id = $6, author_id = $7,
			is_published = $8, meta_title = $9, meta_description = $10, seo = $11, order_index = $12,
			updated_at = $13
		WHERE id = $14
		RETURNING ` + newsColumns

	seoJSON, err := marshalSeo(news.Seo())
	if err != nil {
		return nil, fmt.Errorf("news repository update (marshal seo): %w", err)
	}

	imagesJSON, err := media.MarshalImages(news.Images())
	if err != nil {
		return nil, fmt.Errorf("news repository update (marshal images): %w", err)
	}

	row := tx.QueryRow(ctx, query,
		news.Title(), news.Slug(), imagesJSON, news.Content(), news.Excerpt(), news.CategoryID(), news.AuthorID(),
		news.IsPublished(), news.MetaTitle(), news.MetaDescription(), seoJSON, news.OrderIndex(),
		news.UpdatedAt(), news.ID(),
	)
	updated, err := scanNews(row)
	if err != nil {
		return nil, fmt.Errorf("news repository update: %w", err)
	}

	if tagIDs != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM news_tags WHERE news_id = $1", updated.ID()); err != nil {
			return nil, fmt.Errorf("news repository update (clear tags): %w", err)
		}
		if err := insertNewsTags(ctx, tx, updated.ID(), *tagIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("news repository update (commit tx): %w", err)
	}

	if err := r.attachTags(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *PostgresNewsRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE news SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("news repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresNewsRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE news SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("news repository restore: %w", err)
	}
	return nil
}
