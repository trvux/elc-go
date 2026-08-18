package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/page/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type PostgresPageRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPageRepository(pool *pgxpool.Pool) *PostgresPageRepository {
	return &PostgresPageRepository{pool: pool}
}

func (r *PostgresPageRepository) GetAll(ctx context.Context, filter domain.PageFilter) ([]*domain.Page, error) {
	query := `
		SELECT id, title, slug, content, is_published, meta_title, meta_description, order_index, created_at, updated_at, deleted_at
		FROM pages
		WHERE 1=1`
	args := []any{}
	placeholderIdx := 1

	if !filter.IncludeDeleted {
		query += " AND deleted_at IS NULL"
	}

	if filter.IsPublished != nil {
		query += fmt.Sprintf(" AND is_published = $%d", placeholderIdx)
		args = append(args, *filter.IsPublished)
		placeholderIdx++
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND title ILIKE $%d", placeholderIdx)
		args = append(args, "%"+filter.Search+"%")
		placeholderIdx++
	}

	query += " ORDER BY order_index ASC, created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", placeholderIdx)
		args = append(args, filter.Limit)
		placeholderIdx++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", placeholderIdx)
		args = append(args, filter.Offset)
		placeholderIdx++
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("page repository getAll: %w", err)
	}
	defer rows.Close()

	var pages []*domain.Page
	for rows.Next() {
		p, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("page repository getAll rows: %w", err)
	}

	return pages, nil
}

func (r *PostgresPageRepository) Count(ctx context.Context, filter domain.PageFilter) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM pages
		WHERE 1=1`
	args := []any{}
	placeholderIdx := 1

	if !filter.IncludeDeleted {
		query += " AND deleted_at IS NULL"
	}

	if filter.IsPublished != nil {
		query += fmt.Sprintf(" AND is_published = $%d", placeholderIdx)
		args = append(args, *filter.IsPublished)
		placeholderIdx++
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND title ILIKE $%d", placeholderIdx)
		args = append(args, "%"+filter.Search+"%")
		placeholderIdx++
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("page repository count: %w", err)
	}

	return count, nil
}

func (r *PostgresPageRepository) GetByID(ctx context.Context, id string) (*domain.Page, error) {
	query := `
		SELECT id, title, slug, content, is_published, meta_title, meta_description, order_index, created_at, updated_at, deleted_at
		FROM pages
		WHERE id = $1 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, query, id)
	p, err := r.scanRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *PostgresPageRepository) GetBySlug(ctx context.Context, slug string) (*domain.Page, error) {
	query := `
		SELECT id, title, slug, content, is_published, meta_title, meta_description, order_index, created_at, updated_at, deleted_at
		FROM pages
		WHERE slug = $1 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, query, slug)
	p, err := r.scanRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *PostgresPageRepository) Create(ctx context.Context, p *domain.Page) (*domain.Page, error) {
	query := `
		INSERT INTO pages (title, slug, content, is_published, meta_title, meta_description, order_index, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	var id string
	var createdAt, updatedAt time.Time

	err := r.pool.QueryRow(ctx, query,
		p.Title(), p.Slug(), p.Content(), p.IsPublished(), p.MetaTitle(), p.MetaDescription(), p.OrderIndex(),
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("page repository create: %w", err)
	}

	return domain.RehydratePage(
		id, p.Title(), p.Slug(), p.Content(), p.IsPublished(), p.MetaTitle(), p.MetaDescription(), p.OrderIndex(),
		createdAt, updatedAt, nil,
	), nil
}

func (r *PostgresPageRepository) Update(ctx context.Context, p *domain.Page) (*domain.Page, error) {
	query := `
		UPDATE pages
		SET title = $1, slug = $2, content = $3, is_published = $4, meta_title = $5, meta_description = $6, order_index = $7, updated_at = NOW()
		WHERE id = $8 AND deleted_at IS NULL
		RETURNING updated_at`

	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, query,
		p.Title(), p.Slug(), p.Content(), p.IsPublished(), p.MetaTitle(), p.MetaDescription(), p.OrderIndex(), p.ID(),
	).Scan(&updatedAt)
	if err != nil {
		// The application layer already checked existence via GetByID before
		// calling Update, but the row can still be deleted in between (TOCTOU).
		// Map that case to a proper 404 instead of leaking a raw 500.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NewNotFoundError("page")
		}
		return nil, fmt.Errorf("page repository update: %w", err)
	}

	return domain.RehydratePage(
		p.ID(), p.Title(), p.Slug(), p.Content(), p.IsPublished(), p.MetaTitle(), p.MetaDescription(), p.OrderIndex(),
		p.CreatedAt(), updatedAt, nil,
	), nil
}

func (r *PostgresPageRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE pages SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("page repository delete: %w", err)
	}
	return nil
}

func (r *PostgresPageRepository) Restore(ctx context.Context, id string) error {
	query := `UPDATE pages SET deleted_at = NULL WHERE id = $1 AND deleted_at IS NOT NULL`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("page repository restore: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *PostgresPageRepository) scanRow(scanner rowScanner) (*domain.Page, error) {
	var id string
	var title string
	var slug string
	var content json.RawMessage
	var isPublished bool
	var metaTitle *string
	var metaDescription *string
	var orderIndex int
	var createdAt time.Time
	var updatedAt time.Time
	var deletedAt *time.Time

	err := scanner.Scan(
		&id, &title, &slug, &content, &isPublished, &metaTitle, &metaDescription, &orderIndex, &createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	return domain.RehydratePage(
		id, title, slug, content, isPublished, metaTitle, metaDescription, orderIndex, createdAt, updatedAt, deletedAt,
	), nil
}
