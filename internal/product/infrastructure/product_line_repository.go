package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/product/domain"
)

// PostgresProductLineRepository is a separate small repository (own struct,
// own file) — ProductLine is a simple independent CRUD lookup, same shape
// as brand/category's own repositories, not something that needs to share
// PostgresProductRepository's transaction/variant-tree machinery.
type PostgresProductLineRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ProductLineRepository = (*PostgresProductLineRepository)(nil)

func NewPostgresProductLineRepository(pool *pgxpool.Pool) *PostgresProductLineRepository {
	return &PostgresProductLineRepository{pool: pool}
}

const productLineColumns = `id, brand_id, category_id, code, name, tier_rank, description, mpn_prefixes, created_at, updated_at, deleted_at`

func (r *PostgresProductLineRepository) List(ctx context.Context, brandID *string, includeDeleted bool) ([]*domain.ProductLine, error) {
	query := "SELECT " + productLineColumns + " FROM product_lines"
	conditions := []string{}
	args := []any{}
	if !includeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if brandID != nil {
		args = append(args, *brandID)
		conditions = append(conditions, fmt.Sprintf("brand_id = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE "
		for i, c := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += c
		}
	}
	query += " ORDER BY tier_rank ASC, name ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("product line repository list: %w", err)
	}
	defer rows.Close()

	var result []*domain.ProductLine
	for rows.Next() {
		l, err := scanProductLine(rows)
		if err != nil {
			return nil, fmt.Errorf("product line repository list scan: %w", err)
		}
		result = append(result, l)
	}
	return result, rows.Err()
}

func (r *PostgresProductLineRepository) GetByID(ctx context.Context, id string) (*domain.ProductLine, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+productLineColumns+" FROM product_lines WHERE id = $1 AND deleted_at IS NULL", id)
	l, err := scanProductLine(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("product line repository getById: %w", err)
	}
	return l, nil
}

func (r *PostgresProductLineRepository) Create(ctx context.Context, line *domain.ProductLine) (*domain.ProductLine, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO product_lines (brand_id, category_id, code, name, tier_rank, description, mpn_prefixes)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING `+productLineColumns,
		line.BrandID(), line.CategoryID(), line.Code(), line.Name(), line.TierRank(), line.Description(), orEmptyStrings(line.MpnPrefixes()),
	)
	created, err := scanProductLine(row)
	if err != nil {
		return nil, fmt.Errorf("product line repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresProductLineRepository) Update(ctx context.Context, line *domain.ProductLine) (*domain.ProductLine, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE product_lines
		SET category_id = $1, name = $2, tier_rank = $3, description = $4, mpn_prefixes = $5, updated_at = $6
		WHERE id = $7
		RETURNING `+productLineColumns,
		line.CategoryID(), line.Name(), line.TierRank(), line.Description(), orEmptyStrings(line.MpnPrefixes()), time.Now(), line.ID(),
	)
	updated, err := scanProductLine(row)
	if err != nil {
		return nil, fmt.Errorf("product line repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresProductLineRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE product_lines SET deleted_at = $1 WHERE id = $2", time.Now(), id)
	if err != nil {
		return fmt.Errorf("product line repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresProductLineRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE product_lines SET deleted_at = NULL WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("product line repository restore: %w", err)
	}
	return nil
}

func scanProductLine(row rowScanner) (*domain.ProductLine, error) {
	var (
		id, brandID, code, name string
		categoryID              *string
		tierRank                int
		description             *string
		mpnPrefixes             []string
		createdAt, updatedAt    time.Time
		deletedAt               *time.Time
	)
	if err := row.Scan(&id, &brandID, &categoryID, &code, &name, &tierRank, &description, &mpnPrefixes, &createdAt, &updatedAt, &deletedAt); err != nil {
		return nil, err
	}
	return domain.RehydrateProductLine(id, brandID, categoryID, code, name, tierRank, description, mpnPrefixes, createdAt, updatedAt, deletedAt), nil
}
