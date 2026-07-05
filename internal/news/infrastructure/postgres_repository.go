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
)

type PostgresNewsRepository struct {
	pool *pgxpool.Pool
}

var _ domain.NewsRepository = (*PostgresNewsRepository)(nil)

func NewPostgresNewsRepository(pool *pgxpool.Pool) *PostgresNewsRepository {
	return &PostgresNewsRepository{pool: pool}
}

const newsColumns = "id, title, slug, image, content, category_id, is_published, meta_title, meta_description, order_index, created_at, updated_at, deleted_at"

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNews(row rowScanner) (*domain.News, error) {
	var (
		id, title, slug, image     string
		content                    json.RawMessage
		categoryID                 *string
		isPublished                bool
		metaTitle, metaDescription *string
		orderIndex                 int
		createdAt, updatedAt       time.Time
		deletedAt                  *time.Time
	)

	if err := row.Scan(
		&id, &title, &slug, &image, &content, &categoryID,
		&isPublished, &metaTitle, &metaDescription, &orderIndex,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateNews(
		id, title, slug, image, content, categoryID,
		isPublished, metaTitle, metaDescription, orderIndex,
		createdAt, updatedAt, deletedAt,
	), nil
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
	return n, nil
}

// Create resurrects a soft-deleted row sharing the same slug rather than
// plain-INSERTing — news.slug is a plain UNIQUE constraint, so a plain
// INSERT would fail outright on slug reuse. See domain/repository.go and
// docs/news.md.
func (r *PostgresNewsRepository) Create(ctx context.Context, news *domain.News) (*domain.News, error) {
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

	var row pgx.Row
	if isResurrect {
		query := `
			UPDATE news
			SET title = $1, slug = $2, image = $3, content = $4, category_id = $5,
				is_published = $6, meta_title = $7, meta_description = $8, order_index = $9,
				deleted_at = NULL, updated_at = $10
			WHERE id = $11
			RETURNING ` + newsColumns
		row = tx.QueryRow(ctx, query,
			news.Title(), news.Slug(), news.Image(), news.Content(), news.CategoryID(),
			news.IsPublished(), news.MetaTitle(), news.MetaDescription(), news.OrderIndex(),
			time.Now(), existingID,
		)
	} else {
		query := `
			INSERT INTO news (title, slug, image, content, category_id, is_published, meta_title, meta_description, order_index)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING ` + newsColumns
		row = tx.QueryRow(ctx, query,
			news.Title(), news.Slug(), news.Image(), news.Content(), news.CategoryID(),
			news.IsPublished(), news.MetaTitle(), news.MetaDescription(), news.OrderIndex(),
		)
	}

	created, err := scanNews(row)
	if err != nil {
		return nil, fmt.Errorf("news repository create: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("news repository create (commit tx): %w", err)
	}
	return created, nil
}

func (r *PostgresNewsRepository) Update(ctx context.Context, news *domain.News) (*domain.News, error) {
	query := `
		UPDATE news
		SET title = $1, slug = $2, image = $3, content = $4, category_id = $5,
			is_published = $6, meta_title = $7, meta_description = $8, order_index = $9,
			updated_at = $10
		WHERE id = $11
		RETURNING ` + newsColumns

	row := r.pool.QueryRow(ctx, query,
		news.Title(), news.Slug(), news.Image(), news.Content(), news.CategoryID(),
		news.IsPublished(), news.MetaTitle(), news.MetaDescription(), news.OrderIndex(),
		news.UpdatedAt(), news.ID(),
	)
	updated, err := scanNews(row)
	if err != nil {
		return nil, fmt.Errorf("news repository update: %w", err)
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
