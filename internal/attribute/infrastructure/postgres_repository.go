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

const attributeDefinitionColumns = `id, category_id, code, name, group_label, data_type, unit, options, order_index, is_required, created_at, updated_at, deleted_at`

func (r *PostgresAttributeDefinitionRepository) GetAll(ctx context.Context, filter domain.AttributeDefinitionFilter) ([]*domain.AttributeDefinition, error) {
	query := "SELECT " + attributeDefinitionColumns + " FROM attribute_definitions"
	conditions := []string{}
	args := []any{}
	argN := 1

	if !filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filter.CategoryID != nil {
		if filter.IncludeGlobal {
			conditions = append(conditions, fmt.Sprintf("(category_id = $%d OR category_id IS NULL)", argN))
		} else {
			conditions = append(conditions, fmt.Sprintf("category_id = $%d", argN))
		}
		args = append(args, *filter.CategoryID)
		argN++
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY group_label NULLS FIRST, order_index ASC"

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
	return defs, nil
}

func (r *PostgresAttributeDefinitionRepository) GetByID(ctx context.Context, id string) (*domain.AttributeDefinition, error) {
	query := "SELECT " + attributeDefinitionColumns + " FROM attribute_definitions WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	d, err := scanAttributeDefinition(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("attribute definition repository getById: %w", err)
	}
	return d, nil
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
		INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING ` + attributeDefinitionColumns

	row := r.pool.QueryRow(ctx, query,
		def.CategoryID(), def.Code(), def.Name(), def.GroupLabel(), def.DataType(), def.Unit(),
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
		categoryID, groupLabel   *string
		unit                     *string
		options                  []string
		orderIndex               int
		isRequired               bool
		createdAt, updatedAt     time.Time
		deletedAt                *time.Time
	)

	if err := row.Scan(
		&id, &categoryID, &code, &name, &groupLabel, &dataType, &unit, &options, &orderIndex, &isRequired,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateAttributeDefinition(
		id, categoryID, code, name, groupLabel, dataType, unit, options, orderIndex, isRequired,
		createdAt, updatedAt, deletedAt,
	), nil
}
