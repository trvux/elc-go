package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/category/domain"
)

type PostgresCategoryRepository struct {
	pool *pgxpool.Pool
}

var _ domain.CategoryRepository = (*PostgresCategoryRepository)(nil)

func NewPostgresCategoryRepository(pool *pgxpool.Pool) *PostgresCategoryRepository {
	return &PostgresCategoryRepository{pool: pool}
}

// categoryColumns (unaliased) is used for plain INSERT/UPDATE ... RETURNING
// against the categories table directly. categoryColumnsAliased/categoryJoin
// (with a "c" alias) are used for the read queries that LEFT JOIN
// group_categories.
const categoryColumns = `id, name, slug, group_id, image_url, meta_title, meta_description,
	is_featured, order_index, content, created_at, updated_at, deleted_at`

const categoryColumnsAliased = `c.id, c.name, c.slug, c.group_id, c.image_url, c.meta_title, c.meta_description,
	c.is_featured, c.order_index, c.content, c.created_at, c.updated_at, c.deleted_at`

const categoryJoin = `SELECT ` + categoryColumnsAliased + `,
	g.id, g.name, g.slug, g.image_url, g.meta_title, g.meta_description, g.is_featured, g.order_index
	FROM categories c LEFT JOIN group_categories g ON g.id = c.group_id`

func (r *PostgresCategoryRepository) GetAll(ctx context.Context, filter domain.CategoryFilter) ([]*domain.CategoryWithRelations, error) {
	query := categoryJoin
	conditions := []string{}
	args := []any{}
	argN := 1

	if !filter.IncludeDeleted {
		conditions = append(conditions, "c.deleted_at IS NULL")
	}
	if filter.GroupID != "" {
		conditions = append(conditions, fmt.Sprintf("c.group_id = $%d", argN))
		args = append(args, filter.GroupID)
		argN++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("c.name ILIKE $%d", argN))
		args = append(args, "%"+filter.Search+"%")
		argN++
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY c.order_index ASC, c.name ASC"

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
		return nil, fmt.Errorf("category repository getAll: %w", err)
	}
	defer rows.Close()

	var categories []*domain.CategoryWithRelations
	for rows.Next() {
		c, err := scanCategoryWithRelations(rows)
		if err != nil {
			return nil, fmt.Errorf("category repository getAll scan: %w", err)
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("category repository getAll rows: %w", err)
	}

	return categories, nil
}

func (r *PostgresCategoryRepository) Count(ctx context.Context, filter domain.CategoryFilter) (int, error) {
	query := "SELECT COUNT(*) FROM categories c"
	conditions := []string{}
	args := []any{}
	argN := 1

	if !filter.IncludeDeleted {
		conditions = append(conditions, "c.deleted_at IS NULL")
	}
	if filter.GroupID != "" {
		conditions = append(conditions, fmt.Sprintf("c.group_id = $%d", argN))
		args = append(args, filter.GroupID)
		argN++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("c.name ILIKE $%d", argN))
		args = append(args, "%"+filter.Search+"%")
		argN++
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("category repository count: %w", err)
	}
	return count, nil
}

func (r *PostgresCategoryRepository) GetByID(ctx context.Context, id string) (*domain.CategoryWithRelations, error) {
	query := categoryJoin + " WHERE c.id = $1 AND c.deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	c, err := scanCategoryWithRelations(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("category repository getById: %w", err)
	}
	return c, nil
}

func (r *PostgresCategoryRepository) GetBySlug(ctx context.Context, slug string) (*domain.CategoryWithRelations, error) {
	query := categoryJoin + " WHERE c.slug = $1 AND c.deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	c, err := scanCategoryWithRelations(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("category repository getBySlug: %w", err)
	}
	return c, nil
}

func (r *PostgresCategoryRepository) Create(ctx context.Context, category *domain.Category) (*domain.Category, error) {
	// Resurrect-on-create logic: check if there's an existing soft-deleted category with same slug
	var existingID string
	err := r.pool.QueryRow(ctx,
		"SELECT id FROM categories WHERE slug = $1 AND deleted_at IS NOT NULL",
		category.Slug(),
	).Scan(&existingID)

	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("category repository create (check existing): %w", err)
	}
	isResurrect := (err == nil)

	if isResurrect {
		query := `
			UPDATE categories
			SET name = $1, group_id = $2, image_url = $3, meta_title = $4, meta_description = $5,
				is_featured = $6, order_index = $7, content = $8,
				deleted_at = NULL, updated_at = $9
			WHERE id = $10
			RETURNING ` + categoryColumns

		row := r.pool.QueryRow(ctx, query,
			category.Name(), category.GroupID(), category.ImageURL(), category.MetaTitle(), category.MetaDescription(),
			category.IsFeatured(), category.OrderIndex(), category.Content(),
			time.Now(), existingID,
		)
		resurrected, err := scanCategory(row)
		if err != nil {
			return nil, fmt.Errorf("category repository create (resurrect): %w", err)
		}
		return resurrected, nil
	}

	query := `
		INSERT INTO categories (name, slug, group_id, image_url, meta_title, meta_description, is_featured, order_index, content)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + categoryColumns

	row := r.pool.QueryRow(ctx, query,
		category.Name(), category.Slug(), category.GroupID(), category.ImageURL(), category.MetaTitle(), category.MetaDescription(),
		category.IsFeatured(), category.OrderIndex(), category.Content(),
	)
	created, err := scanCategory(row)
	if err != nil {
		return nil, fmt.Errorf("category repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresCategoryRepository) Update(ctx context.Context, category *domain.Category) (*domain.Category, error) {
	query := `
		UPDATE categories
		SET name = $1, slug = $2, group_id = $3, image_url = $4, meta_title = $5, meta_description = $6,
			is_featured = $7, order_index = $8, content = $9, updated_at = $10
		WHERE id = $11
		RETURNING ` + categoryColumns

	row := r.pool.QueryRow(ctx, query,
		category.Name(), category.Slug(), category.GroupID(), category.ImageURL(), category.MetaTitle(), category.MetaDescription(),
		category.IsFeatured(), category.OrderIndex(), category.Content(),
		category.UpdatedAt(), category.ID(),
	)
	updated, err := scanCategory(row)
	if err != nil {
		return nil, fmt.Errorf("category repository update: %w", err)
	}
	return updated, nil
}

// SoftDelete transactionally soft-deletes the category and hard-deletes its
// project_type_category associations — fixes a bug in the old Next.js code,
// which ran these as two separate, non-transactional queries. category has no
// child rows to cascade into (unlike group, which cascades into categories),
// so this is a single-ID version of group's SoftDelete — see
// internal/group/infrastructure/postgres_repository.go.
func (r *PostgresCategoryRepository) SoftDelete(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("category repository softDelete (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	if _, err := tx.Exec(ctx,
		"UPDATE categories SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		now, id,
	); err != nil {
		return fmt.Errorf("category repository softDelete (categories): %w", err)
	}

	if _, err := tx.Exec(ctx,
		"DELETE FROM project_type_category WHERE category_id = $1",
		id,
	); err != nil {
		return fmt.Errorf("category repository softDelete (project_type_category): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("category repository softDelete (commit tx): %w", err)
	}
	return nil
}

func (r *PostgresCategoryRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE categories SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("category repository restore: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

// scanCategory scans a bare `categoryColumns` row (no group join) — used by
// Create/Update, which RETURNING only the categories table's own columns.
func scanCategory(row rowScanner) (*domain.Category, error) {
	var (
		id, name, slug                       string
		groupID                              *string
		imageUrl, metaTitle, metaDescription *string
		isFeatured                           bool
		orderIndex                           int
		content                              json.RawMessage
		createdAt, updatedAt                 time.Time
		deletedAt                            *time.Time
	)

	if err := row.Scan(
		&id, &name, &slug, &groupID, &imageUrl, &metaTitle, &metaDescription,
		&isFeatured, &orderIndex, &content, &createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateCategory(
		id, name, slug, groupID, imageUrl, metaTitle, metaDescription,
		isFeatured, orderIndex, content, createdAt, updatedAt, deletedAt,
	), nil
}

// scanCategoryWithRelations scans a `categoryJoin` row (category columns +
// nullable group_categories columns) — used by GetAll/GetByID/GetBySlug.
func scanCategoryWithRelations(row rowScanner) (*domain.CategoryWithRelations, error) {
	var (
		id, name, slug                       string
		groupID                              *string
		imageUrl, metaTitle, metaDescription *string
		isFeatured                           bool
		orderIndex                           int
		content                              json.RawMessage
		createdAt, updatedAt                 time.Time
		deletedAt                            *time.Time

		gID, gName, gSlug                       *string
		gImageUrl, gMetaTitle, gMetaDescription *string
		gIsFeatured                             *bool
		gOrderIndex                             *int
	)

	if err := row.Scan(
		&id, &name, &slug, &groupID, &imageUrl, &metaTitle, &metaDescription,
		&isFeatured, &orderIndex, &content, &createdAt, &updatedAt, &deletedAt,
		&gID, &gName, &gSlug, &gImageUrl, &gMetaTitle, &gMetaDescription, &gIsFeatured, &gOrderIndex,
	); err != nil {
		return nil, err
	}

	category := domain.RehydrateCategory(
		id, name, slug, groupID, imageUrl, metaTitle, metaDescription,
		isFeatured, orderIndex, content, createdAt, updatedAt, deletedAt,
	)

	var group *domain.GroupRef
	if gID != nil {
		group = &domain.GroupRef{
			ID:              *gID,
			Name:            derefString(gName),
			Slug:            derefString(gSlug),
			ImageURL:        gImageUrl,
			MetaTitle:       gMetaTitle,
			MetaDescription: gMetaDescription,
			IsFeatured:      gIsFeatured != nil && *gIsFeatured,
			OrderIndex:      derefInt(gOrderIndex),
		}
	}

	return &domain.CategoryWithRelations{Category: category, Group: group}, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
