package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/tag/domain"
)

type PostgresTagRepository struct {
	pool *pgxpool.Pool
}

var _ domain.TagRepository = (*PostgresTagRepository)(nil)

func NewPostgresTagRepository(pool *pgxpool.Pool) *PostgresTagRepository {
	return &PostgresTagRepository{pool: pool}
}

const tagColumns = `id, name, slug, created_at, updated_at, deleted_at`

func (r *PostgresTagRepository) GetAll(ctx context.Context, filter domain.TagFilter) ([]*domain.Tag, error) {
	query := "SELECT " + tagColumns + " FROM tags"
	conditions := []string{}
	args := []any{}
	argN := 1

	if !filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argN))
		args = append(args, "%"+filter.Search+"%")
		argN++
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY name ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, filter.Limit)
		argN++
		query += fmt.Sprintf(" OFFSET $%d", argN)
		args = append(args, filter.Offset)
		argN++
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("tag repository getAll: %w", err)
	}
	defer rows.Close()

	var tags []*domain.Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, fmt.Errorf("tag repository getAll scan: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tag repository getAll rows: %w", err)
	}

	return tags, nil
}

func (r *PostgresTagRepository) GetByID(ctx context.Context, id string) (*domain.Tag, error) {
	query := "SELECT " + tagColumns + " FROM tags WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	t, err := scanTag(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("tag repository getById: %w", err)
	}
	return t, nil
}

func (r *PostgresTagRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tag, error) {
	query := "SELECT " + tagColumns + " FROM tags WHERE slug = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	t, err := scanTag(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("tag repository getBySlug: %w", err)
	}
	return t, nil
}

func (r *PostgresTagRepository) GetByIDs(ctx context.Context, ids []string) ([]*domain.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query := "SELECT " + tagColumns + " FROM tags WHERE id = ANY($1) AND deleted_at IS NULL ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("tag repository getByIds: %w", err)
	}
	defer rows.Close()

	var tags []*domain.Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, fmt.Errorf("tag repository getByIds scan: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tag repository getByIds rows: %w", err)
	}
	return tags, nil
}

func (r *PostgresTagRepository) Create(ctx context.Context, tag *domain.Tag) (*domain.Tag, error) {
	query := `
		INSERT INTO tags (name, slug)
		VALUES ($1, $2)
		RETURNING ` + tagColumns

	row := r.pool.QueryRow(ctx, query, tag.Name(), tag.Slug())
	created, err := scanTag(row)
	if err != nil {
		return nil, fmt.Errorf("tag repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresTagRepository) Update(ctx context.Context, tag *domain.Tag) (*domain.Tag, error) {
	query := `
		UPDATE tags
		SET name = $1, slug = $2, updated_at = $3
		WHERE id = $4
		RETURNING ` + tagColumns

	row := r.pool.QueryRow(ctx, query, tag.Name(), tag.Slug(), tag.UpdatedAt(), tag.ID())
	updated, err := scanTag(row)
	if err != nil {
		return nil, fmt.Errorf("tag repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresTagRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE tags SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("tag repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresTagRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE tags SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("tag repository restore: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTag(row rowScanner) (*domain.Tag, error) {
	var (
		id, name, slug       string
		createdAt, updatedAt time.Time
		deletedAt            *time.Time
	)

	if err := row.Scan(&id, &name, &slug, &createdAt, &updatedAt, &deletedAt); err != nil {
		return nil, err
	}

	return domain.RehydrateTag(id, name, slug, createdAt, updatedAt, deletedAt), nil
}
