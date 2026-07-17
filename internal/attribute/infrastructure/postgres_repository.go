package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/attribute/domain"
)

type PostgresAttributeDefinitionRepository struct {
	pool *pgxpool.Pool
}

var _ domain.AttributeDefinitionRepository = (*PostgresAttributeDefinitionRepository)(nil)

func NewPostgresAttributeDefinitionRepository(pool *pgxpool.Pool) *PostgresAttributeDefinitionRepository {
	return &PostgresAttributeDefinitionRepository{pool: pool}
}

const attributeDefinitionColumns = `id, code, name, group_label, data_type, unit, options, order_index, is_required, created_at, updated_at, deleted_at`

func (r *PostgresAttributeDefinitionRepository) GetAll(ctx context.Context, filter domain.AttributeDefinitionFilter) ([]*domain.AttributeDefinitionWithCategories, error) {
	query := "SELECT " + attributeDefinitionColumns + " FROM attribute_definitions ad"
	conditions := []string{}
	args := []any{}
	argN := 1

	if !filter.IncludeDeleted {
		conditions = append(conditions, "ad.deleted_at IS NULL")
	}
	if filter.CategoryID != nil {
		inCategory := fmt.Sprintf("EXISTS (SELECT 1 FROM category_attribute_definitions cad WHERE cad.attribute_definition_id = ad.id AND cad.category_id = $%d)", argN)
		if filter.IncludeGlobal {
			conditions = append(conditions, fmt.Sprintf("(%s OR NOT EXISTS (SELECT 1 FROM category_attribute_definitions g WHERE g.attribute_definition_id = ad.id))", inCategory))
		} else {
			conditions = append(conditions, inCategory)
		}
		args = append(args, *filter.CategoryID)
		argN++
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY ad.group_label NULLS FIRST, ad.order_index ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("attribute definition repository getAll: %w", err)
	}
	defer rows.Close()

	var defs []*domain.AttributeDefinition
	for rows.Next() {
		d, err := scanAttributeDefinition(rows)
		if err != nil {
			return nil, fmt.Errorf("attribute definition repository getAll scan: %w", err)
		}
		defs = append(defs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attribute definition repository getAll rows: %w", err)
	}
	return r.attachCategoryIDs(ctx, defs)
}

func (r *PostgresAttributeDefinitionRepository) GetByID(ctx context.Context, id string) (*domain.AttributeDefinitionWithCategories, error) {
	query := "SELECT " + attributeDefinitionColumns + " FROM attribute_definitions WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	d, err := scanAttributeDefinition(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("attribute definition repository getById: %w", err)
	}
	withCategories, err := r.attachCategoryIDs(ctx, []*domain.AttributeDefinition{d})
	if err != nil {
		return nil, err
	}
	return withCategories[0], nil
}

func (r *PostgresAttributeDefinitionRepository) GetByIDs(ctx context.Context, ids []string) ([]*domain.AttributeDefinition, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query := "SELECT " + attributeDefinitionColumns + " FROM attribute_definitions WHERE id = ANY($1) AND deleted_at IS NULL"

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("attribute definition repository getByIds: %w", err)
	}
	defer rows.Close()

	var defs []*domain.AttributeDefinition
	for rows.Next() {
		d, err := scanAttributeDefinition(rows)
		if err != nil {
			return nil, fmt.Errorf("attribute definition repository getByIds scan: %w", err)
		}
		defs = append(defs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attribute definition repository getByIds rows: %w", err)
	}
	return defs, nil
}

func (r *PostgresAttributeDefinitionRepository) Create(ctx context.Context, def *domain.AttributeDefinition) (*domain.AttributeDefinition, error) {
	query := `
		INSERT INTO attribute_definitions (code, name, group_label, data_type, unit, options, order_index, is_required)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING ` + attributeDefinitionColumns

	row := r.pool.QueryRow(ctx, query,
		def.Code(), def.Name(), def.GroupLabel(), def.DataType(), def.Unit(),
		orEmptyStrings(def.Options()), def.OrderIndex(), def.IsRequired(),
	)
	created, err := scanAttributeDefinition(row)
	if err != nil {
		return nil, fmt.Errorf("attribute definition repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresAttributeDefinitionRepository) Update(ctx context.Context, def *domain.AttributeDefinition) (*domain.AttributeDefinition, error) {
	query := `
		UPDATE attribute_definitions
		SET name = $1, group_label = $2, unit = $3, options = $4, order_index = $5, is_required = $6, updated_at = $7
		WHERE id = $8
		RETURNING ` + attributeDefinitionColumns

	row := r.pool.QueryRow(ctx, query,
		def.Name(), def.GroupLabel(), def.Unit(), orEmptyStrings(def.Options()), def.OrderIndex(), def.IsRequired(),
		def.UpdatedAt(), def.ID(),
	)
	updated, err := scanAttributeDefinition(row)
	if err != nil {
		return nil, fmt.Errorf("attribute definition repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresAttributeDefinitionRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE attribute_definitions SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("attribute definition repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresAttributeDefinitionRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE attribute_definitions SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("attribute definition repository restore: %w", err)
	}
	return nil
}

func (r *PostgresAttributeDefinitionRepository) AttachCategories(ctx context.Context, definitionID string, categoryIDs []string) error {
	if len(categoryIDs) == 0 {
		return nil
	}
	args := []any{}
	placeholders := make([]string, len(categoryIDs))
	for i, categoryID := range categoryIDs {
		placeholders[i] = fmt.Sprintf("($%d,$%d)", len(args)+1, len(args)+2)
		args = append(args, categoryID, definitionID)
	}
	query := "INSERT INTO category_attribute_definitions (category_id, attribute_definition_id) VALUES " +
		strings.Join(placeholders, ",") + " ON CONFLICT DO NOTHING"
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("attribute definition repository attachCategories: %w", err)
	}
	return nil
}

func (r *PostgresAttributeDefinitionRepository) DetachCategory(ctx context.Context, definitionID, categoryID string) error {
	_, err := r.pool.Exec(ctx,
		"DELETE FROM category_attribute_definitions WHERE attribute_definition_id = $1 AND category_id = $2",
		definitionID, categoryID,
	)
	if err != nil {
		return fmt.Errorf("attribute definition repository detachCategory: %w", err)
	}
	return nil
}

func (r *PostgresAttributeDefinitionRepository) GetApplicableForCategory(ctx context.Context, categoryID string) ([]*domain.AttributeDefinition, error) {
	query := `
		SELECT ` + attributeDefinitionColumns + ` FROM attribute_definitions ad
		WHERE ad.deleted_at IS NULL
		AND (
			EXISTS (SELECT 1 FROM category_attribute_definitions cad WHERE cad.attribute_definition_id = ad.id AND cad.category_id = $1)
			OR NOT EXISTS (SELECT 1 FROM category_attribute_definitions g WHERE g.attribute_definition_id = ad.id)
		)
		ORDER BY ad.group_label NULLS FIRST, ad.order_index ASC`

	rows, err := r.pool.Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("attribute definition repository getApplicableForCategory: %w", err)
	}
	defer rows.Close()

	var defs []*domain.AttributeDefinition
	for rows.Next() {
		d, err := scanAttributeDefinition(rows)
		if err != nil {
			return nil, fmt.Errorf("attribute definition repository getApplicableForCategory scan: %w", err)
		}
		defs = append(defs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attribute definition repository getApplicableForCategory rows: %w", err)
	}
	return defs, nil
}

// attachCategoryIDs batch-loads category_attribute_definitions rows for the
// given definitions and wraps each in AttributeDefinitionWithCategories —
// same "one extra query, group in Go" shape as product's attachVariantTree.
func (r *PostgresAttributeDefinitionRepository) attachCategoryIDs(ctx context.Context, defs []*domain.AttributeDefinition) ([]*domain.AttributeDefinitionWithCategories, error) {
	result := make([]*domain.AttributeDefinitionWithCategories, len(defs))
	ids := make([]string, len(defs))
	for i, d := range defs {
		ids[i] = d.ID()
	}

	byDefID := map[string][]string{}
	if len(ids) > 0 {
		rows, err := r.pool.Query(ctx,
			"SELECT attribute_definition_id, category_id FROM category_attribute_definitions WHERE attribute_definition_id = ANY($1)",
			ids,
		)
		if err != nil {
			return nil, fmt.Errorf("attribute definition repository attachCategoryIDs: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var defID, categoryID string
			if err := rows.Scan(&defID, &categoryID); err != nil {
				return nil, fmt.Errorf("attribute definition repository attachCategoryIDs scan: %w", err)
			}
			byDefID[defID] = append(byDefID[defID], categoryID)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("attribute definition repository attachCategoryIDs rows: %w", err)
		}
	}

	for i, d := range defs {
		result[i] = &domain.AttributeDefinitionWithCategories{
			AttributeDefinition: d,
			CategoryIDs:         byDefID[d.ID()],
		}
	}
	return result, nil
}

// orEmptyStrings defaults a nil slice to an empty (non-nil) one before
// binding to a text[] column — same helper product's postgres_repository.go
// already has for images/labels, duplicated here rather than exported
// cross-module for a one-line helper.
func orEmptyStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAttributeDefinition(row rowScanner) (*domain.AttributeDefinition, error) {
	var (
		id, code, name, dataType string
		groupLabel               *string
		unit                     *string
		options                  []string
		orderIndex               int
		isRequired               bool
		createdAt, updatedAt     time.Time
		deletedAt                *time.Time
	)

	if err := row.Scan(
		&id, &code, &name, &groupLabel, &dataType, &unit, &options, &orderIndex, &isRequired,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateAttributeDefinition(
		id, code, name, groupLabel, dataType, unit, options, orderIndex, isRequired,
		createdAt, updatedAt, deletedAt,
	), nil
}
