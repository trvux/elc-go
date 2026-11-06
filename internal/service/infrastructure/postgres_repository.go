package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/service/domain"
)

type PostgresServiceRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ServiceRepository = (*PostgresServiceRepository)(nil)

func NewPostgresServiceRepository(pool *pgxpool.Pool) *PostgresServiceRepository {
	return &PostgresServiceRepository{pool: pool}
}

const serviceColumns = `s.id, s.title, s.title_align, s.slug, s.group_id, s.labels,
	s.line, s.brief_prompts,
	s.description, s.content, s.images, s.meta_title, s.meta_description,
	s.is_published, s.order_index,
	s.created_at, s.updated_at, s.deleted_at,
	g.id, g.name`

// selectWithRelations LEFT JOINs only the 2 fields (id, name) any UI actually
// reads from the service group.
const selectWithRelations = `
	SELECT ` + serviceColumns + `
	FROM services s
	LEFT JOIN service_groups g ON g.id = s.group_id`

func (r *PostgresServiceRepository) Count(ctx context.Context, filter domain.ServiceFilter) (int, error) {
	query := "SELECT COUNT(*) FROM services s"
	conditions, args := buildFilterConditions(filter)
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("service repository count: %w", err)
	}
	return count, nil
}

func (r *PostgresServiceRepository) GetAll(ctx context.Context, filter domain.ServiceFilter) ([]*domain.ServiceWithRelations, error) {
	query := selectWithRelations
	conditions, args := buildFilterConditions(filter)
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY s.order_index ASC, s.created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("service repository getAll: %w", err)
	}
	defer rows.Close()

	var services []*domain.ServiceWithRelations
	for rows.Next() {
		s, err := scanServiceWithRelations(rows)
		if err != nil {
			return nil, fmt.Errorf("service repository getAll scan: %w", err)
		}
		services = append(services, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("service repository getAll rows: %w", err)
	}

	return services, nil
}

func (r *PostgresServiceRepository) GetByID(ctx context.Context, id string) (*domain.ServiceWithRelations, error) {
	query := selectWithRelations + " WHERE s.id = $1 AND s.deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	s, err := scanServiceWithRelations(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("service repository getById: %w", err)
	}
	return s, nil
}

func (r *PostgresServiceRepository) GetBySlug(ctx context.Context, slug string) (*domain.ServiceWithRelations, error) {
	query := selectWithRelations + " WHERE s.slug = $1 AND s.deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	s, err := scanServiceWithRelations(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("service repository getBySlug: %w", err)
	}
	return s, nil
}

func (r *PostgresServiceRepository) Create(ctx context.Context, service *domain.Service) (*domain.Service, error) {
	query := `
		INSERT INTO services (
			title, title_align, slug, group_id, labels, line, brief_prompts, description, content, images,
			meta_title, meta_description, is_published, order_index
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, title, title_align, slug, group_id, labels, line, brief_prompts, description, content, images,
			meta_title, meta_description, is_published, order_index, created_at, updated_at, deleted_at`

	imagesJSON, err := media.MarshalImages(service.Images())
	if err != nil {
		return nil, fmt.Errorf("service repository create (marshal images): %w", err)
	}

	row := r.pool.QueryRow(ctx, query,
		service.Title(), service.TitleAlign(), service.Slug(), service.GroupID(),
		service.Labels(), service.Line(), service.BriefPrompts(),
		service.Description(), service.Content(), imagesJSON,
		service.MetaTitle(), service.MetaDescription(), service.IsPublished(), service.OrderIndex(),
	)
	created, err := scanService(row)
	if err != nil {
		return nil, fmt.Errorf("service repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresServiceRepository) Update(ctx context.Context, service *domain.Service) (*domain.Service, error) {
	query := `
		UPDATE services
		SET title = $1, title_align = $2, slug = $3, group_id = $4, labels = $5,
			line = $6, brief_prompts = $7, description = $8, content = $9, images = $10,
			meta_title = $11, meta_description = $12, is_published = $13, order_index = $14, updated_at = $15
		WHERE id = $16
		RETURNING id, title, title_align, slug, group_id, labels, line, brief_prompts, description, content, images,
			meta_title, meta_description, is_published, order_index, created_at, updated_at, deleted_at`

	imagesJSON, err := media.MarshalImages(service.Images())
	if err != nil {
		return nil, fmt.Errorf("service repository update (marshal images): %w", err)
	}

	row := r.pool.QueryRow(ctx, query,
		service.Title(), service.TitleAlign(), service.Slug(), service.GroupID(),
		service.Labels(), service.Line(), service.BriefPrompts(),
		service.Description(), service.Content(), imagesJSON,
		service.MetaTitle(), service.MetaDescription(), service.IsPublished(), service.OrderIndex(),
		service.UpdatedAt(), service.ID(),
	)
	updated, err := scanService(row)
	if err != nil {
		return nil, fmt.Errorf("service repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresServiceRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE services SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("service repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresServiceRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE services SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("service repository restore: %w", err)
	}
	return nil
}

func buildFilterConditions(filter domain.ServiceFilter) ([]string, []any) {
	conditions := []string{}
	args := []any{}
	argN := 1

	if !filter.IncludeDeleted {
		conditions = append(conditions, "s.deleted_at IS NULL")
	}
	if filter.GroupID != nil {
		conditions = append(conditions, fmt.Sprintf("s.group_id = $%d", argN))
		args = append(args, *filter.GroupID)
		argN++
	}
	if filter.IsPublished != nil {
		conditions = append(conditions, fmt.Sprintf("s.is_published = $%d", argN))
		args = append(args, *filter.IsPublished)
		argN++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("s.title ILIKE $%d", argN))
		args = append(args, "%"+filter.Search+"%")
		argN++
	}

	return conditions, args
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanService(row rowScanner) (*domain.Service, error) {
	var (
		id, title, titleAlign, slug string
		groupID                     *string
		description                 *string
		metaTitle, metaDescription  *string
		labels, briefPrompts        []string
		line                        string
		content                     json.RawMessage
		imagesRaw                   []byte
		isPublished                 bool
		orderIndex                  int
		createdAt, updatedAt        time.Time
		deletedAt                   *time.Time
	)

	if err := row.Scan(
		&id, &title, &titleAlign, &slug, &groupID, &labels,
		&line, &briefPrompts,
		&description, &content, &imagesRaw, &metaTitle, &metaDescription,
		&isPublished, &orderIndex,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	images, err := media.UnmarshalImages(imagesRaw)
	if err != nil {
		return nil, fmt.Errorf("scan service (unmarshal images): %w", err)
	}

	return domain.RehydrateService(
		id, title, titleAlign, slug, groupID, labels, line, briefPrompts, description, content,
		images, metaTitle, metaDescription, isPublished, orderIndex,
		createdAt, updatedAt, deletedAt,
	), nil
}

func scanServiceWithRelations(row rowScanner) (*domain.ServiceWithRelations, error) {
	var (
		id, title, titleAlign, slug string
		groupID                     *string
		description                 *string
		metaTitle, metaDescription  *string
		labels, briefPrompts        []string
		line                        string
		content                     json.RawMessage
		imagesRaw                   []byte
		isPublished                 bool
		orderIndex                  int
		createdAt, updatedAt        time.Time
		deletedAt                   *time.Time
		groupRefID, groupRefName    *string
	)

	if err := row.Scan(
		&id, &title, &titleAlign, &slug, &groupID, &labels,
		&line, &briefPrompts,
		&description, &content, &imagesRaw, &metaTitle, &metaDescription,
		&isPublished, &orderIndex,
		&createdAt, &updatedAt, &deletedAt,
		&groupRefID, &groupRefName,
	); err != nil {
		return nil, err
	}

	images, err := media.UnmarshalImages(imagesRaw)
	if err != nil {
		return nil, fmt.Errorf("scan service with relations (unmarshal images): %w", err)
	}

	service := domain.RehydrateService(
		id, title, titleAlign, slug, groupID, labels, line, briefPrompts, description, content,
		images, metaTitle, metaDescription, isPublished, orderIndex,
		createdAt, updatedAt, deletedAt,
	)

	result := &domain.ServiceWithRelations{Service: service}
	if groupRefID != nil {
		result.Group = &domain.GroupRef{ID: *groupRefID, Name: *groupRefName}
	}

	return result, nil
}
