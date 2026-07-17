package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/group/domain"
)

type PostgresGroupRepository struct {
	pool *pgxpool.Pool
}

var _ domain.GroupRepository = (*PostgresGroupRepository)(nil)

func NewPostgresGroupRepository(pool *pgxpool.Pool) *PostgresGroupRepository {
	return &PostgresGroupRepository{pool: pool}
}

const groupColumns = `id, name, slug, image_url, meta_title, meta_description,
	is_featured, is_hidden, order_index, content, created_at, updated_at, deleted_at`

func (r *PostgresGroupRepository) GetAll(ctx context.Context, filter domain.GroupFilter) ([]*domain.Group, error) {
	query := "SELECT " + groupColumns + " FROM group_categories"
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
		return nil, fmt.Errorf("group repository getAll: %w", err)
	}
	defer rows.Close()

	var groups []*domain.Group
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("group repository getAll scan: %w", err)
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("group repository getAll rows: %w", err)
	}

	return groups, nil
}

func (r *PostgresGroupRepository) GetByID(ctx context.Context, id string) (*domain.Group, error) {
	query := "SELECT " + groupColumns + " FROM group_categories WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	g, err := scanGroup(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("group repository getById: %w", err)
	}
	return g, nil
}

func (r *PostgresGroupRepository) GetBySlug(ctx context.Context, slug string) (*domain.Group, error) {
	query := "SELECT " + groupColumns + " FROM group_categories WHERE slug = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	g, err := scanGroup(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("group repository getBySlug: %w", err)
	}
	return g, nil
}

func (r *PostgresGroupRepository) Create(ctx context.Context, group *domain.Group) (*domain.Group, error) {
	// Resurrect-on-create logic: check if there's an existing soft-deleted group with same slug
	var existingID string
	err := r.pool.QueryRow(ctx,
		"SELECT id FROM group_categories WHERE slug = $1 AND deleted_at IS NOT NULL",
		group.Slug(),
	).Scan(&existingID)

	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("group repository create (check existing): %w", err)
	}
	isResurrect := (err == nil)

	if isResurrect {
		// Existing soft-deleted row found, resurrect it by updating it and setting deleted_at = NULL
		query := `
			UPDATE group_categories
			SET name = $1, image_url = $2, meta_title = $3, meta_description = $4,
				is_featured = $5, is_hidden = $6, order_index = $7, content = $8,
				deleted_at = NULL, updated_at = $9
			WHERE id = $10
			RETURNING ` + groupColumns

		row := r.pool.QueryRow(ctx, query,
			group.Name(), group.ImageURL(), group.MetaTitle(), group.MetaDescription(),
			group.IsFeatured(), group.IsHidden(), group.OrderIndex(), group.Content(),
			time.Now(), existingID,
		)
		resurrected, err := scanGroup(row)
		if err != nil {
			return nil, fmt.Errorf("group repository create (resurrect): %w", err)
		}
		return resurrected, nil
	}

	// No soft-deleted row, insert as new
	query := `
		INSERT INTO group_categories (name, slug, image_url, meta_title, meta_description, is_featured, is_hidden, order_index, content)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + groupColumns

	row := r.pool.QueryRow(ctx, query,
		group.Name(), group.Slug(), group.ImageURL(), group.MetaTitle(), group.MetaDescription(),
		group.IsFeatured(), group.IsHidden(), group.OrderIndex(), group.Content(),
	)
	created, err := scanGroup(row)
	if err != nil {
		return nil, fmt.Errorf("group repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresGroupRepository) Update(ctx context.Context, group *domain.Group) (*domain.Group, error) {
	query := `
		UPDATE group_categories
		SET name = $1, slug = $2, image_url = $3, meta_title = $4, meta_description = $5,
			is_featured = $6, is_hidden = $7, order_index = $8, content = $9, updated_at = $10
		WHERE id = $11
		RETURNING ` + groupColumns

	row := r.pool.QueryRow(ctx, query,
		group.Name(), group.Slug(), group.ImageURL(), group.MetaTitle(), group.MetaDescription(),
		group.IsFeatured(), group.IsHidden(), group.OrderIndex(), group.Content(),
		group.UpdatedAt(), group.ID(),
	)
	updated, err := scanGroup(row)
	if err != nil {
		return nil, fmt.Errorf("group repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresGroupRepository) SoftDelete(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("group repository softDelete (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// 1. Soft delete the group category itself
	if _, err := tx.Exec(ctx,
		"UPDATE group_categories SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		now, id,
	); err != nil {
		return fmt.Errorf("group repository softDelete (group_categories): %w", err)
	}

	// 2. Fetch all active categories under this group category
	rows, err := tx.Query(ctx,
		"SELECT id FROM categories WHERE group_id = $1 AND deleted_at IS NULL",
		id,
	)
	if err != nil {
		return fmt.Errorf("group repository softDelete (fetch categories): %w", err)
	}
	defer rows.Close()

	var categoryIDs []string
	for rows.Next() {
		var catID string
		if err := rows.Scan(&catID); err != nil {
			return fmt.Errorf("group repository softDelete (scan category id): %w", err)
		}
		categoryIDs = append(categoryIDs, catID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("group repository softDelete (categories rows err): %w", err)
	}
	rows.Close() // Close early before Exec calls

	if len(categoryIDs) > 0 {
		// 3. Soft delete those categories
		if _, err := tx.Exec(ctx,
			"UPDATE categories SET deleted_at = $1, updated_at = $1 WHERE id = ANY($2)",
			now, categoryIDs,
		); err != nil {
			return fmt.Errorf("group repository softDelete (soft delete categories): %w", err)
		}

		// 4. Hard delete referencing rows in project_type_category join table
		if _, err := tx.Exec(ctx,
			"DELETE FROM project_type_category WHERE category_id = ANY($1)",
			categoryIDs,
		); err != nil {
			return fmt.Errorf("group repository softDelete (project_type_category): %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("group repository softDelete (commit tx): %w", err)
	}
	return nil
}

func (r *PostgresGroupRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE group_categories SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("group repository restore: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanGroup(row rowScanner) (*domain.Group, error) {
	var (
		id, name, slug                       string
		imageUrl, metaTitle, metaDescription *string
		isFeatured                           bool
		isHidden                             bool
		orderIndex                           int
		content                              json.RawMessage
		createdAt, updatedAt                 time.Time
		deletedAt                            *time.Time
	)

	if err := row.Scan(
		&id, &name, &slug, &imageUrl, &metaTitle, &metaDescription,
		&isFeatured, &isHidden, &orderIndex, &content, &createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateGroup(
		id, name, slug, imageUrl, metaTitle, metaDescription,
		isFeatured, isHidden, orderIndex, content, createdAt, updatedAt, deletedAt,
	), nil
}
