package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/system-page/domain"
)

const systemPageColumns = `id, name, slug, meta_title, meta_description, created_at, updated_at`

type PostgresSystemPageRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSystemPageRepository(pool *pgxpool.Pool) *PostgresSystemPageRepository {
	return &PostgresSystemPageRepository{pool: pool}
}

func (r *PostgresSystemPageRepository) GetAll(ctx context.Context) ([]*domain.SystemPage, error) {
	query := `SELECT ` + systemPageColumns + ` FROM system_pages ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query system pages: %w", err)
	}
	defer rows.Close()

	var pages []*domain.SystemPage
	for rows.Next() {
		p, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during system pages rows iteration: %w", err)
	}

	return pages, nil
}

func (r *PostgresSystemPageRepository) GetByID(ctx context.Context, id string) (*domain.SystemPage, error) {
	query := `SELECT ` + systemPageColumns + ` FROM system_pages WHERE id = $1`

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

func (r *PostgresSystemPageRepository) GetBySlug(ctx context.Context, slug string) (*domain.SystemPage, error) {
	query := `SELECT ` + systemPageColumns + ` FROM system_pages WHERE slug = $1`

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

func (r *PostgresSystemPageRepository) Update(ctx context.Context, p *domain.SystemPage) (*domain.SystemPage, error) {
	query := `
		UPDATE system_pages
		SET meta_title = $1, meta_description = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at`

	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, query, p.MetaTitle(), p.MetaDescription(), p.ID()).Scan(&updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update system page: %w", err)
	}

	return domain.RehydrateSystemPage(
		p.ID(), p.Name(), p.Slug(), p.MetaTitle(), p.MetaDescription(), p.CreatedAt(), updatedAt,
	), nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *PostgresSystemPageRepository) scanRow(scanner rowScanner) (*domain.SystemPage, error) {
	var id, name, slug string
	var metaTitle, metaDescription *string
	var createdAt, updatedAt time.Time

	err := scanner.Scan(&id, &name, &slug, &metaTitle, &metaDescription, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	return domain.RehydrateSystemPage(id, name, slug, metaTitle, metaDescription, createdAt, updatedAt), nil
}
