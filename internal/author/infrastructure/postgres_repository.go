package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/author/domain"
)

type PostgresAuthorRepository struct {
	pool *pgxpool.Pool
}

var _ domain.AuthorRepository = (*PostgresAuthorRepository)(nil)

func NewPostgresAuthorRepository(pool *pgxpool.Pool) *PostgresAuthorRepository {
	return &PostgresAuthorRepository{pool: pool}
}

const authorColumns = `id, name, slug, avatar_url, bio, created_at, updated_at, deleted_at`

func (r *PostgresAuthorRepository) GetAll(ctx context.Context, filter domain.AuthorFilter) ([]*domain.Author, error) {
	query := "SELECT " + authorColumns + " FROM authors"
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
		return nil, fmt.Errorf("author repository getAll: %w", err)
	}
	defer rows.Close()

	var authors []*domain.Author
	for rows.Next() {
		a, err := scanAuthor(rows)
		if err != nil {
			return nil, fmt.Errorf("author repository getAll scan: %w", err)
		}
		authors = append(authors, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("author repository getAll rows: %w", err)
	}

	return authors, nil
}

func (r *PostgresAuthorRepository) GetByID(ctx context.Context, id string) (*domain.Author, error) {
	query := "SELECT " + authorColumns + " FROM authors WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	a, err := scanAuthor(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("author repository getById: %w", err)
	}
	return a, nil
}

func (r *PostgresAuthorRepository) GetBySlug(ctx context.Context, slug string) (*domain.Author, error) {
	query := "SELECT " + authorColumns + " FROM authors WHERE slug = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	a, err := scanAuthor(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("author repository getBySlug: %w", err)
	}
	return a, nil
}

func (r *PostgresAuthorRepository) Create(ctx context.Context, author *domain.Author) (*domain.Author, error) {
	query := `
		INSERT INTO authors (name, slug, avatar_url, bio)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + authorColumns

	row := r.pool.QueryRow(ctx, query, author.Name(), author.Slug(), author.AvatarURL(), author.Bio())
	created, err := scanAuthor(row)
	if err != nil {
		return nil, fmt.Errorf("author repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresAuthorRepository) Update(ctx context.Context, author *domain.Author) (*domain.Author, error) {
	query := `
		UPDATE authors
		SET name = $1, slug = $2, avatar_url = $3, bio = $4, updated_at = $5
		WHERE id = $6
		RETURNING ` + authorColumns

	row := r.pool.QueryRow(ctx, query,
		author.Name(), author.Slug(), author.AvatarURL(), author.Bio(),
		author.UpdatedAt(), author.ID(),
	)
	updated, err := scanAuthor(row)
	if err != nil {
		return nil, fmt.Errorf("author repository update: %w", err)
	}
	return updated, nil
}

// SoftDelete only touches the authors row — news.author_id's FK is ON DELETE
// SET NULL, but that only fires on a real DELETE, not this UPDATE, so
// existing articles keep pointing at the now-soft-deleted author (same
// "leftover FK reference on soft-delete" precedent as brand/products).
func (r *PostgresAuthorRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE authors SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("author repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresAuthorRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE authors SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("author repository restore: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAuthor(row rowScanner) (*domain.Author, error) {
	var (
		id, name, slug, avatarURL, bio string
		createdAt, updatedAt           time.Time
		deletedAt                      *time.Time
	)

	if err := row.Scan(&id, &name, &slug, &avatarURL, &bio, &createdAt, &updatedAt, &deletedAt); err != nil {
		return nil, err
	}

	return domain.RehydrateAuthor(id, name, slug, avatarURL, bio, createdAt, updatedAt, deletedAt), nil
}
