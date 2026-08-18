package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/project-type/domain"
)

type PostgresProjectTypeRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ProjectTypeRepository = (*PostgresProjectTypeRepository)(nil)

func NewPostgresProjectTypeRepository(pool *pgxpool.Pool) *PostgresProjectTypeRepository {
	return &PostgresProjectTypeRepository{pool: pool}
}

const projectTypeColumns = `id, name, slug, image, meta_title, meta_description,
	is_featured, order_index, created_at, updated_at, deleted_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func scanProjectType(row rowScanner) (*domain.ProjectType, error) {
	var (
		id, name, slug             string
		image, metaTitle, metaDesc *string
		isFeatured                 bool
		orderIndex                 int
		createdAt, updatedAt       time.Time
		deletedAt                  *time.Time
	)

	if err := row.Scan(
		&id, &name, &slug, &image, &metaTitle, &metaDesc,
		&isFeatured, &orderIndex, &createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateProjectType(
		id, name, slug, image, metaTitle, metaDesc,
		isFeatured, orderIndex, createdAt, updatedAt, deletedAt,
	), nil
}

func (r *PostgresProjectTypeRepository) GetAll(ctx context.Context, filter domain.ProjectTypeFilter) ([]*domain.ProjectTypeWithCategories, error) {
	conditions, args := buildProjectTypeConditions(filter)

	query := "SELECT " + projectTypeColumns + " FROM project_type"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY order_index ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("project type repository getAll: %w", err)
	}
	defer rows.Close()

	var projectTypes []*domain.ProjectType
	var ids []string
	for rows.Next() {
		pt, err := scanProjectType(rows)
		if err != nil {
			return nil, fmt.Errorf("project type repository getAll scan: %w", err)
		}
		projectTypes = append(projectTypes, pt)
		ids = append(ids, pt.ID())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("project type repository getAll rows: %w", err)
	}

	categoriesByType, err := r.fetchCategoriesForProjectTypes(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ProjectTypeWithCategories, 0, len(projectTypes))
	for _, pt := range projectTypes {
		result = append(result, &domain.ProjectTypeWithCategories{
			ProjectType: pt,
			Categories:  categoriesByType[pt.ID()],
		})
	}
	return result, nil
}

func (r *PostgresProjectTypeRepository) Count(ctx context.Context, filter domain.ProjectTypeFilter) (int, error) {
	conditions, args := buildProjectTypeConditions(filter)
	query := "SELECT COUNT(*) FROM project_type"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("project type repository count: %w", err)
	}
	return count, nil
}

func buildProjectTypeConditions(filter domain.ProjectTypeFilter) ([]string, []any) {
	var conditions []string
	var args []any
	argN := 1
	next := func() int {
		n := argN
		argN++
		return n
	}

	if !filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", next()))
		args = append(args, "%"+filter.Search+"%")
	}

	return conditions, args
}

func (r *PostgresProjectTypeRepository) GetByID(ctx context.Context, id string) (*domain.ProjectTypeWithCategories, error) {
	query := "SELECT " + projectTypeColumns + " FROM project_type WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	pt, err := scanProjectType(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("project type repository getById: %w", err)
	}

	categoriesByType, err := r.fetchCategoriesForProjectTypes(ctx, []string{pt.ID()})
	if err != nil {
		return nil, err
	}
	return &domain.ProjectTypeWithCategories{ProjectType: pt, Categories: categoriesByType[pt.ID()]}, nil
}

// fetchCategoriesForProjectTypes batch-loads project_type_category rows (+
// category + its group) for every project type id in one round trip, keyed
// by project type id — avoids the N+1 the naive per-row version would cause,
// same pattern internal/project's fetchCategoriesForProjects uses. Soft-
// deleted categories are filtered out at the JOIN (c.deleted_at IS NULL)
// rather than post-filtering in application code, unlike the old TS
// mapToDomainWithCategories which filtered in JS.
func (r *PostgresProjectTypeRepository) fetchCategoriesForProjectTypes(ctx context.Context, projectTypeIDs []string) (map[string][]domain.CategoryRef, error) {
	if len(projectTypeIDs) == 0 {
		return map[string][]domain.CategoryRef{}, nil
	}

	query := `
		SELECT ptc.project_type_id,
			c.id, c.name, c.group_id, c.slug, c.image_url, c.meta_title, c.meta_description,
			c.is_featured, c.order_index, c.created_at, c.updated_at, c.deleted_at,
			gc.id, gc.name, gc.slug, gc.image_url, gc.meta_title, gc.meta_description, gc.is_featured, gc.order_index
		FROM project_type_category ptc
		JOIN categories c ON c.id = ptc.category_id AND c.deleted_at IS NULL
		LEFT JOIN group_categories gc ON gc.id = c.group_id
		WHERE ptc.project_type_id = ANY($1)
		ORDER BY c.name ASC`

	rows, err := r.pool.Query(ctx, query, projectTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("project type repository fetchCategories: %w", err)
	}
	defer rows.Close()

	result := map[string][]domain.CategoryRef{}
	for rows.Next() {
		var (
			projectTypeID, catID, catName           string
			groupID                                 *string
			catSlug, catImageURL                    *string
			catMetaTitle, catMetaDescription        *string
			catIsFeatured                           bool
			catOrderIndex                           int
			catCreatedAt, catUpdatedAt              time.Time
			catDeletedAt                            *time.Time
			gID, gName, gSlug                       *string
			gImageURL, gMetaTitle, gMetaDescription *string
			gIsFeatured                             *bool
			gOrderIndex                             *int
		)
		if err := rows.Scan(
			&projectTypeID, &catID, &catName, &groupID, &catSlug, &catImageURL, &catMetaTitle, &catMetaDescription,
			&catIsFeatured, &catOrderIndex, &catCreatedAt, &catUpdatedAt, &catDeletedAt,
			&gID, &gName, &gSlug, &gImageURL, &gMetaTitle, &gMetaDescription, &gIsFeatured, &gOrderIndex,
		); err != nil {
			return nil, fmt.Errorf("project type repository fetchCategories scan: %w", err)
		}

		var group *domain.CategoryGroupRef
		if gID != nil {
			group = &domain.CategoryGroupRef{
				ID:              *gID,
				Name:            derefStr(gName),
				Slug:            derefStr(gSlug),
				ImageURL:        gImageURL,
				MetaTitle:       gMetaTitle,
				MetaDescription: gMetaDescription,
				IsFeatured:      gIsFeatured != nil && *gIsFeatured,
				OrderIndex:      derefInt(gOrderIndex),
			}
		}

		result[projectTypeID] = append(result[projectTypeID], domain.CategoryRef{
			ID: catID, Name: catName, GroupID: groupID, Slug: derefStr(catSlug),
			ImageURL: catImageURL, MetaTitle: catMetaTitle, MetaDescription: catMetaDescription,
			IsFeatured: catIsFeatured, OrderIndex: catOrderIndex,
			CreatedAt: catCreatedAt, UpdatedAt: catUpdatedAt, DeletedAt: catDeletedAt,
			Group: group,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("project type repository fetchCategories rows: %w", err)
	}
	return result, nil
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// Create resurrects a soft-deleted row sharing the same slug rather than
// plain-INSERTing — project_type.slug is a plain UNIQUE constraint (see
// migrations/000001_baseline_project_type.up.sql), so a plain INSERT would
// fail the constraint outright on slug reuse. Both the resurrect-or-insert
// step and the relation writes run in one transaction — the old TS create()
// ran the equivalent as several sequential, un-transactioned Supabase calls
// (find-existing, insert/update, delete old relations, insert new
// relations). See docs/project-type.md.
func (r *PostgresProjectTypeRepository) Create(ctx context.Context, pt *domain.ProjectType, categoryIDs []string) (*domain.ProjectType, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("project type repository create (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	// FOR UPDATE locks the soft-deleted row (if any) for the rest of this
	// transaction so a second concurrent Create for the same slug blocks
	// here instead of racing this check against the UPDATE below.
	var existingID string
	err = tx.QueryRow(ctx,
		"SELECT id FROM project_type WHERE slug = $1 AND deleted_at IS NOT NULL FOR UPDATE",
		pt.Slug(),
	).Scan(&existingID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("project type repository create (check existing): %w", err)
	}
	isResurrect := err == nil

	var createdID string
	if isResurrect {
		query := `
			UPDATE project_type
			SET name = $1, image = $2, meta_title = $3, meta_description = $4,
				is_featured = $5, order_index = $6, deleted_at = NULL, updated_at = $7
			WHERE id = $8
			RETURNING id`
		if err := tx.QueryRow(ctx, query,
			pt.Name(), pt.Image(), pt.MetaTitle(), pt.MetaDescription(),
			pt.IsFeatured(), pt.OrderIndex(), time.Now(), existingID,
		).Scan(&createdID); err != nil {
			return nil, fmt.Errorf("project type repository create (resurrect): %w", err)
		}

		// Old TS create()'s resurrect path always starts a resurrected
		// project type's relations fresh rather than merging with whatever
		// the soft-deleted row used to have.
		if _, err := tx.Exec(ctx, "DELETE FROM project_type_category WHERE project_type_id = $1", createdID); err != nil {
			return nil, fmt.Errorf("project type repository create (clear categories): %w", err)
		}
	} else {
		query := `
			INSERT INTO project_type (name, slug, image, meta_title, meta_description, is_featured, order_index)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`
		if err := tx.QueryRow(ctx, query,
			pt.Name(), pt.Slug(), pt.Image(), pt.MetaTitle(), pt.MetaDescription(),
			pt.IsFeatured(), pt.OrderIndex(),
		).Scan(&createdID); err != nil {
			return nil, fmt.Errorf("project type repository create: %w", err)
		}
	}

	if err := insertProjectTypeCategories(ctx, tx, createdID, categoryIDs); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, "SELECT "+projectTypeColumns+" FROM project_type WHERE id = $1", createdID)
	created, err := scanProjectType(row)
	if err != nil {
		return nil, fmt.Errorf("project type repository create (reload): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("project type repository create (commit tx): %w", err)
	}
	return created, nil
}

// Update runs the row update + full relation replace in one transaction,
// same atomicity fix as Create — see its doc comment.
func (r *PostgresProjectTypeRepository) Update(ctx context.Context, pt *domain.ProjectType, categoryIDs *[]string) (*domain.ProjectType, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("project type repository update (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE project_type
		SET name = $1, slug = $2, image = $3, meta_title = $4, meta_description = $5,
			is_featured = $6, order_index = $7, updated_at = $8
		WHERE id = $9
		RETURNING ` + projectTypeColumns

	row := tx.QueryRow(ctx, query,
		pt.Name(), pt.Slug(), pt.Image(), pt.MetaTitle(), pt.MetaDescription(),
		pt.IsFeatured(), pt.OrderIndex(), pt.UpdatedAt(), pt.ID(),
	)
	updated, err := scanProjectType(row)
	if err != nil {
		return nil, fmt.Errorf("project type repository update: %w", err)
	}

	if categoryIDs != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM project_type_category WHERE project_type_id = $1", pt.ID()); err != nil {
			return nil, fmt.Errorf("project type repository update (clear categories): %w", err)
		}
		if err := insertProjectTypeCategories(ctx, tx, pt.ID(), *categoryIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("project type repository update (commit tx): %w", err)
	}
	return updated, nil
}

func insertProjectTypeCategories(ctx context.Context, tx pgx.Tx, projectTypeID string, categoryIDs []string) error {
	if len(categoryIDs) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO project_type_category (project_type_id, category_id) VALUES ")
	args := make([]any, 0, len(categoryIDs)*2)
	argN := 1
	for i, catID := range categoryIDs {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("($%d, $%d)", argN, argN+1))
		args = append(args, projectTypeID, catID)
		argN += 2
	}
	if _, err := tx.Exec(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("project type repository insertCategories: %w", err)
	}
	return nil
}

// SoftDelete mirrors the old TS delete() exactly: soft-delete the
// project_type row, null out project_type_id on referencing projects (the
// FK's own ON DELETE SET NULL never fires because this is an UPDATE, not a
// real DELETE), and hard-delete this project type's project_type_category
// rows — all three in one transaction, fixing the old TS's un-transactioned
// three-call version. See modules/project-type/infrastructure/projectTypeRepo.ts.
func (r *PostgresProjectTypeRepository) SoftDelete(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("project type repository softDelete (begin tx): %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	if _, err := tx.Exec(ctx,
		"UPDATE project_type SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		now, id,
	); err != nil {
		return fmt.Errorf("project type repository softDelete (project_type): %w", err)
	}

	if _, err := tx.Exec(ctx,
		"UPDATE projects SET project_type_id = NULL WHERE project_type_id = $1",
		id,
	); err != nil {
		return fmt.Errorf("project type repository softDelete (projects): %w", err)
	}

	if _, err := tx.Exec(ctx,
		"DELETE FROM project_type_category WHERE project_type_id = $1",
		id,
	); err != nil {
		return fmt.Errorf("project type repository softDelete (project_type_category): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("project type repository softDelete (commit tx): %w", err)
	}
	return nil
}
