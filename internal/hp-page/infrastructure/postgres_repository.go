package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/hp-page/domain"
)

type PostgresHpPageRepository struct {
	pool *pgxpool.Pool
}

var _ domain.HpPageRepository = (*PostgresHpPageRepository)(nil)

func NewPostgresHpPageRepository(pool *pgxpool.Pool) *PostgresHpPageRepository {
	return &PostgresHpPageRepository{pool: pool}
}

const hpPageColumns = `id, name, slug, image_url, meta_title, meta_description,
	order_index, content, attribute_code, attribute_values, created_at, updated_at, deleted_at`

func (r *PostgresHpPageRepository) GetAll(ctx context.Context, filter domain.HpPageFilter) ([]*domain.HpPage, error) {
	query := "SELECT " + hpPageColumns + " FROM hp_pages"
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
	query += " ORDER BY order_index ASC, name ASC"

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
		return nil, fmt.Errorf("hp_page repository getAll: %w", err)
	}
	defer rows.Close()

	var pages []*domain.HpPage
	for rows.Next() {
		p, err := scanHpPage(rows)
		if err != nil {
			return nil, fmt.Errorf("hp_page repository getAll scan: %w", err)
		}
		pages = append(pages, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("hp_page repository getAll rows: %w", err)
	}

	return pages, nil
}

func (r *PostgresHpPageRepository) GetByID(ctx context.Context, id string) (*domain.HpPage, error) {
	query := "SELECT " + hpPageColumns + " FROM hp_pages WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	p, err := scanHpPage(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("hp_page repository getById: %w", err)
	}
	return p, nil
}

func (r *PostgresHpPageRepository) GetBySlug(ctx context.Context, slug string) (*domain.HpPage, error) {
	query := "SELECT " + hpPageColumns + " FROM hp_pages WHERE slug = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	p, err := scanHpPage(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("hp_page repository getBySlug: %w", err)
	}
	return p, nil
}

// Create is a plain INSERT — slug is only unique among non-deleted rows
// (a partial index), so a soft-deleted hp_page never blocks reusing its
// slug and no "resurrect" step is needed here, same as brand.
func (r *PostgresHpPageRepository) Create(ctx context.Context, page *domain.HpPage) (*domain.HpPage, error) {
	query := `
		INSERT INTO hp_pages (name, slug, image_url, meta_title, meta_description, order_index, content, attribute_code, attribute_values)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + hpPageColumns

	row := r.pool.QueryRow(ctx, query,
		page.Name(), page.Slug(), page.ImageURL(), page.MetaTitle(), page.MetaDescription(),
		page.OrderIndex(), page.Content(), page.AttributeCode(), page.AttributeValues(),
	)
	created, err := scanHpPage(row)
	if err != nil {
		return nil, fmt.Errorf("hp_page repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresHpPageRepository) Update(ctx context.Context, page *domain.HpPage) (*domain.HpPage, error) {
	query := `
		UPDATE hp_pages
		SET name = $1, slug = $2, image_url = $3, meta_title = $4, meta_description = $5,
			order_index = $6, content = $7, attribute_code = $8, attribute_values = $9, updated_at = $10
		WHERE id = $11
		RETURNING ` + hpPageColumns

	row := r.pool.QueryRow(ctx, query,
		page.Name(), page.Slug(), page.ImageURL(), page.MetaTitle(), page.MetaDescription(),
		page.OrderIndex(), page.Content(), page.AttributeCode(), page.AttributeValues(),
		page.UpdatedAt(), page.ID(),
	)
	updated, err := scanHpPage(row)
	if err != nil {
		return nil, fmt.Errorf("hp_page repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresHpPageRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE hp_pages SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("hp_page repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresHpPageRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE hp_pages SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("hp_page repository restore: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanHpPage(row rowScanner) (*domain.HpPage, error) {
	var (
		id, name, slug, imageURL   string
		metaTitle, metaDescription *string
		orderIndex                 int
		content                    json.RawMessage
		attributeCode              string
		attributeValues            []string
		createdAt, updatedAt       time.Time
		deletedAt                  *time.Time
	)

	if err := row.Scan(
		&id, &name, &slug, &imageURL, &metaTitle, &metaDescription,
		&orderIndex, &content, &attributeCode, &attributeValues, &createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateHpPage(
		id, name, slug, imageURL, metaTitle, metaDescription,
		orderIndex, content, attributeCode, attributeValues, createdAt, updatedAt, deletedAt,
	), nil
}
