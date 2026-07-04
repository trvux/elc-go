package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/branch/domain"
)

type PostgresBranchRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBranchRepository(pool *pgxpool.Pool) *PostgresBranchRepository {
	return &PostgresBranchRepository{pool: pool}
}

const branchColumns = "id, name, slug, address, phone, email, maps_url, maps_embed, description, image_url, is_published, order_index, meta_title, meta_description, created_at, updated_at, deleted_at"

func (r *PostgresBranchRepository) GetAll(ctx context.Context, filter domain.BranchFilter) ([]*domain.Branch, error) {
	var conditions []string
	var args []any
	placeholderCount := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.IsPublished != nil {
		conditions = append(conditions, fmt.Sprintf("is_published = $%d", placeholderCount))
		args = append(args, *filter.IsPublished)
		placeholderCount++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR address ILIKE $%d)", placeholderCount, placeholderCount))
		args = append(args, "%"+filter.Search+"%")
		placeholderCount++
	}

	query := "SELECT " + branchColumns + " FROM branches"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY order_index ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", placeholderCount)
		args = append(args, filter.Limit)
		placeholderCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", placeholderCount)
		args = append(args, filter.Offset)
		placeholderCount++
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("branch repository getAll: %w", err)
	}
	defer rows.Close()

	var branches []*domain.Branch
	for rows.Next() {
		b, err := scanBranch(rows)
		if err != nil {
			return nil, fmt.Errorf("branch repository getAll (scan): %w", err)
		}
		branches = append(branches, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("branch repository getAll (rows err): %w", err)
	}

	return branches, nil
}

func (r *PostgresBranchRepository) Count(ctx context.Context, filter domain.BranchFilter) (int, error) {
	var conditions []string
	var args []any
	placeholderCount := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.IsPublished != nil {
		conditions = append(conditions, fmt.Sprintf("is_published = $%d", placeholderCount))
		args = append(args, *filter.IsPublished)
		placeholderCount++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR address ILIKE $%d)", placeholderCount, placeholderCount))
		args = append(args, "%"+filter.Search+"%")
		placeholderCount++
	}

	query := "SELECT count(*) FROM branches"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("branch repository count: %w", err)
	}

	return count, nil
}

func (r *PostgresBranchRepository) GetByID(ctx context.Context, id string) (*domain.Branch, error) {
	query := "SELECT " + branchColumns + " FROM branches WHERE id = $1 AND deleted_at IS NULL"
	row := r.pool.QueryRow(ctx, query, id)
	b, err := scanBranch(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("branch repository getByID: %w", err)
	}
	return b, nil
}

func (r *PostgresBranchRepository) GetBySlug(ctx context.Context, slug string) (*domain.Branch, error) {
	query := "SELECT " + branchColumns + " FROM branches WHERE slug = $1 AND deleted_at IS NULL"
	row := r.pool.QueryRow(ctx, query, slug)
	b, err := scanBranch(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("branch repository getBySlug: %w", err)
	}
	return b, nil
}

func (r *PostgresBranchRepository) Create(ctx context.Context, branch *domain.Branch) (*domain.Branch, error) {
	query := `
		INSERT INTO branches (
			name, slug, address, phone, email, maps_url, maps_embed, description,
			image_url, is_published, order_index, meta_title, meta_description
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING ` + branchColumns

	row := r.pool.QueryRow(ctx, query,
		branch.Name(), branch.Slug(), branch.Address(), branch.Phone(), branch.Email(),
		branch.MapsURL(), branch.MapsEmbed(), branch.Description(), branch.ImageUrl(),
		branch.IsPublished(), branch.OrderIndex(), branch.MetaTitle(), branch.MetaDescription(),
	)
	created, err := scanBranch(row)
	if err != nil {
		return nil, fmt.Errorf("branch repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresBranchRepository) Update(ctx context.Context, branch *domain.Branch) (*domain.Branch, error) {
	query := `
		UPDATE branches
		SET name = $1, slug = $2, address = $3, phone = $4, email = $5,
			maps_url = $6, maps_embed = $7, description = $8, image_url = $9,
			is_published = $10, order_index = $11, meta_title = $12, meta_description = $13,
			updated_at = $14
		WHERE id = $15
		RETURNING ` + branchColumns

	row := r.pool.QueryRow(ctx, query,
		branch.Name(), branch.Slug(), branch.Address(), branch.Phone(), branch.Email(),
		branch.MapsURL(), branch.MapsEmbed(), branch.Description(), branch.ImageUrl(),
		branch.IsPublished(), branch.OrderIndex(), branch.MetaTitle(), branch.MetaDescription(),
		branch.UpdatedAt(), branch.ID(),
	)
	updated, err := scanBranch(row)
	if err != nil {
		return nil, fmt.Errorf("branch repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresBranchRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM branches WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("branch repository delete: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBranch(row rowScanner) (*domain.Branch, error) {
	var (
		id, name, slug, address, phone, email, mapsURL, mapsEmbed string
		description json.RawMessage
		imageUrl *string
		isPublished bool
		orderIndex int
		metaTitle, metaDescription *string
		createdAt, updatedAt time.Time
		deletedAt *time.Time
	)

	if err := row.Scan(
		&id, &name, &slug, &address, &phone, &email, &mapsURL, &mapsEmbed, &description,
		&imageUrl, &isPublished, &orderIndex, &metaTitle, &metaDescription,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateBranch(
		id, name, slug, address, phone, email, mapsURL, mapsEmbed, description,
		imageUrl, isPublished, orderIndex, metaTitle, metaDescription,
		createdAt, updatedAt, deletedAt,
	), nil
}
