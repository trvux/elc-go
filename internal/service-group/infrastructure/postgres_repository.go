package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/service-group/domain"
)

type PostgresServiceGroupRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ServiceGroupRepository = (*PostgresServiceGroupRepository)(nil)

func NewPostgresServiceGroupRepository(pool *pgxpool.Pool) *PostgresServiceGroupRepository {
	return &PostgresServiceGroupRepository{pool: pool}
}

const serviceGroupColumns = `id, name, slug, image_url, meta_title, meta_description,
	is_featured, order_index, category_ids, created_at, updated_at, deleted_at`

func (r *PostgresServiceGroupRepository) GetAll(ctx context.Context, filter domain.ServiceGroupFilter) ([]*domain.ServiceGroup, error) {
	query := "SELECT " + serviceGroupColumns + " FROM service_groups"
	conditions := []string{}
	args := []any{}
	argN := 1

	if !filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filter.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("is_featured = $%d", argN))
		args = append(args, *filter.IsFeatured)
		argN++
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY order_index ASC, created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("service group repository getAll: %w", err)
	}
	defer rows.Close()

	var groups []*domain.ServiceGroup
	for rows.Next() {
		sg, err := scanServiceGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("service group repository getAll scan: %w", err)
		}
		groups = append(groups, sg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("service group repository getAll rows: %w", err)
	}

	return groups, nil
}

func (r *PostgresServiceGroupRepository) GetByID(ctx context.Context, id string) (*domain.ServiceGroup, error) {
	query := "SELECT " + serviceGroupColumns + " FROM service_groups WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	sg, err := scanServiceGroup(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("service group repository getById: %w", err)
	}
	return sg, nil
}

func (r *PostgresServiceGroupRepository) GetBySlug(ctx context.Context, slug string) (*domain.ServiceGroup, error) {
	query := "SELECT " + serviceGroupColumns + " FROM service_groups WHERE slug = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	sg, err := scanServiceGroup(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("service group repository getBySlug: %w", err)
	}
	return sg, nil
}

// Create resurrects a soft-deleted row with the same slug instead of
// inserting, because slug is globally unique even for deleted rows — see
// docs/service-group.md.
func (r *PostgresServiceGroupRepository) Create(ctx context.Context, serviceGroup *domain.ServiceGroup) (*domain.ServiceGroup, error) {
	var existingID string
	err := r.pool.QueryRow(ctx,
		"SELECT id FROM service_groups WHERE slug = $1 AND deleted_at IS NOT NULL",
		serviceGroup.Slug(),
	).Scan(&existingID)

	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("service group repository create (check existing): %w", err)
	}

	if err == nil {
		query := `
			UPDATE service_groups
			SET name = $1, image_url = $2, meta_title = $3, meta_description = $4,
				is_featured = $5, order_index = $6, category_ids = $7,
				deleted_at = NULL, updated_at = $8
			WHERE id = $9
			RETURNING ` + serviceGroupColumns

		row := r.pool.QueryRow(ctx, query,
			serviceGroup.Name(), serviceGroup.ImageURL(), serviceGroup.MetaTitle(), serviceGroup.MetaDescription(),
			serviceGroup.IsFeatured(), serviceGroup.OrderIndex(), serviceGroup.CategoryIDs(),
			time.Now(), existingID,
		)
		resurrected, err := scanServiceGroup(row)
		if err != nil {
			return nil, fmt.Errorf("service group repository create (resurrect): %w", err)
		}
		return resurrected, nil
	}

	query := `
		INSERT INTO service_groups (name, slug, image_url, meta_title, meta_description, is_featured, order_index, category_ids)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ` + serviceGroupColumns

	row := r.pool.QueryRow(ctx, query,
		serviceGroup.Name(), serviceGroup.Slug(), serviceGroup.ImageURL(), serviceGroup.MetaTitle(), serviceGroup.MetaDescription(),
		serviceGroup.IsFeatured(), serviceGroup.OrderIndex(), serviceGroup.CategoryIDs(),
	)
	created, err := scanServiceGroup(row)
	if err != nil {
		return nil, fmt.Errorf("service group repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresServiceGroupRepository) Update(ctx context.Context, serviceGroup *domain.ServiceGroup) (*domain.ServiceGroup, error) {
	query := `
		UPDATE service_groups
		SET name = $1, slug = $2, image_url = $3, meta_title = $4, meta_description = $5,
			is_featured = $6, order_index = $7, category_ids = $8, updated_at = $9
		WHERE id = $10
		RETURNING ` + serviceGroupColumns

	row := r.pool.QueryRow(ctx, query,
		serviceGroup.Name(), serviceGroup.Slug(), serviceGroup.ImageURL(), serviceGroup.MetaTitle(), serviceGroup.MetaDescription(),
		serviceGroup.IsFeatured(), serviceGroup.OrderIndex(), serviceGroup.CategoryIDs(),
		serviceGroup.UpdatedAt(), serviceGroup.ID(),
	)
	updated, err := scanServiceGroup(row)
	if err != nil {
		return nil, fmt.Errorf("service group repository update: %w", err)
	}
	return updated, nil
}

// SoftDelete sets deleted_at AND clears group_id on any services that
// belonged to this group — the FK's ON DELETE SET NULL does not fire for a
// soft delete (it's an UPDATE, not a DELETE), so this must be done by hand.
// Both writes run in one transaction so they can't partially apply.
func (r *PostgresServiceGroupRepository) SoftDelete(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("service group repository softDelete (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		"UPDATE service_groups SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	); err != nil {
		return fmt.Errorf("service group repository softDelete: %w", err)
	}

	if _, err := tx.Exec(ctx, "UPDATE services SET group_id = NULL WHERE group_id = $1", id); err != nil {
		return fmt.Errorf("service group repository softDelete (cleanup services): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("service group repository softDelete (commit tx): %w", err)
	}
	return nil
}

func (r *PostgresServiceGroupRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE service_groups SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("service group repository restore: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanServiceGroup(row rowScanner) (*domain.ServiceGroup, error) {
	var (
		id, name, slug                       string
		imageURL, metaTitle, metaDescription *string
		isFeatured                           bool
		orderIndex                           int
		categoryIDs                          []string
		createdAt, updatedAt                 time.Time
		deletedAt                            *time.Time
	)

	if err := row.Scan(
		&id, &name, &slug, &imageURL, &metaTitle, &metaDescription,
		&isFeatured, &orderIndex, &categoryIDs, &createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateServiceGroup(
		id, name, slug, imageURL, metaTitle, metaDescription,
		isFeatured, orderIndex, categoryIDs, createdAt, updatedAt, deletedAt,
	), nil
}
