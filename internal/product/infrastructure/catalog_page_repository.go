package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/product/domain"
)

// PostgresCatalogPageRepository operates on the single product_catalog_page
// row — every query is a bare SELECT/UPDATE with no WHERE clause, since the
// table only ever holds one row (seeded by migration 000012).
type PostgresCatalogPageRepository struct {
	pool *pgxpool.Pool
}

var _ domain.CatalogPageRepository = (*PostgresCatalogPageRepository)(nil)

func NewPostgresCatalogPageRepository(pool *pgxpool.Pool) *PostgresCatalogPageRepository {
	return &PostgresCatalogPageRepository{pool: pool}
}

func (r *PostgresCatalogPageRepository) Get(ctx context.Context) (*domain.CatalogPage, error) {
	row := r.pool.QueryRow(ctx, "SELECT content, meta_title, meta_description, updated_at FROM product_catalog_page LIMIT 1")

	var (
		content                    []byte
		metaTitle, metaDescription *string
		updatedAt                  time.Time
	)
	if err := row.Scan(&content, &metaTitle, &metaDescription, &updatedAt); err != nil {
		return nil, fmt.Errorf("catalog page repository get: %w", err)
	}
	return domain.RehydrateCatalogPage(content, metaTitle, metaDescription, updatedAt), nil
}

func (r *PostgresCatalogPageRepository) Update(ctx context.Context, input domain.UpdateCatalogPageInput) (*domain.CatalogPage, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE product_catalog_page
		SET content = $1, meta_title = $2, meta_description = $3, updated_at = $4
		RETURNING content, meta_title, meta_description, updated_at`,
		input.Content, input.MetaTitle, input.MetaDescription, time.Now(),
	)

	var (
		content                    []byte
		metaTitle, metaDescription *string
		updatedAt                  time.Time
	)
	if err := row.Scan(&content, &metaTitle, &metaDescription, &updatedAt); err != nil {
		return nil, fmt.Errorf("catalog page repository update: %w", err)
	}
	return domain.RehydrateCatalogPage(content, metaTitle, metaDescription, updatedAt), nil
}
