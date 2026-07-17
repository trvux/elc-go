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

const serviceColumns = `s.id, s.title, s.slug, s.group_id, s.category_id,
	s.original_price, s.discount_percent, s.price_display_text, s.labels,
	s.description, s.content, s.images, s.meta_title, s.meta_description,
	s.is_featured, s.is_published, s.order_index,
	s.created_at, s.updated_at, s.deleted_at,
	g.id, g.name, c.id, c.name`

// selectWithRelations LEFT JOINs only the 2 fields (id, name) any UI actually
// reads from group/category — see docs/service.md for why this stops one
// level short of the old TS code's 3-table category.group join.
const selectWithRelations = `
	SELECT ` + serviceColumns + `
	FROM services s
	LEFT JOIN service_groups g ON g.id = s.group_id
	LEFT JOIN categories c ON c.id = s.category_id`

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
			title, slug, group_id, category_id, original_price, sale_price, discount_percent,
			price_display_text, labels, description, content, images, meta_title, meta_description,
			is_featured, is_published, order_index
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, title, slug, group_id, category_id, original_price, discount_percent,
			price_display_text, labels, description, content, images, meta_title, meta_description,
			is_featured, is_published, order_index, created_at, updated_at, deleted_at`

	imagesJSON, err := media.MarshalImages(service.Images())
	if err != nil {
		return nil, fmt.Errorf("service repository create (marshal images): %w", err)
	}

	row := r.pool.QueryRow(ctx, query,
		service.Title(), service.Slug(), service.GroupID(), service.CategoryID(),
		service.OriginalPrice(), service.SalePrice(), service.DiscountPercent(),
		service.PriceDisplayText(), service.Labels(), service.Description(), service.Content(),
		imagesJSON, service.MetaTitle(), service.MetaDescription(),
		service.IsFeatured(), service.IsPublished(), service.OrderIndex(),
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
		SET title = $1, slug = $2, group_id = $3, category_id = $4, original_price = $5,
			sale_price = $6, discount_percent = $7, price_display_text = $8, labels = $9,
			description = $10, content = $11, images = $12, meta_title = $13, meta_description = $14,
			is_featured = $15, is_published = $16, order_index = $17, updated_at = $18
		WHERE id = $19
		RETURNING id, title, slug, group_id, category_id, original_price, discount_percent,
			price_display_text, labels, description, content, images, meta_title, meta_description,
			is_featured, is_published, order_index, created_at, updated_at, deleted_at`

	imagesJSON, err := media.MarshalImages(service.Images())
	if err != nil {
		return nil, fmt.Errorf("service repository update (marshal images): %w", err)
	}

	row := r.pool.QueryRow(ctx, query,
		service.Title(), service.Slug(), service.GroupID(), service.CategoryID(),
		service.OriginalPrice(), service.SalePrice(), service.DiscountPercent(),
		service.PriceDisplayText(), service.Labels(), service.Description(), service.Content(),
		imagesJSON, service.MetaTitle(), service.MetaDescription(),
		service.IsFeatured(), service.IsPublished(), service.OrderIndex(),
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
	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("s.category_id = $%d", argN))
		args = append(args, *filter.CategoryID)
		argN++
	}
	if filter.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("s.is_featured = $%d", argN))
		args = append(args, *filter.IsFeatured)
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
		id, title, slug               string
		groupID, categoryID           *string
		originalPrice                 *int64
		discountPercent               *int
		priceDisplayText, description *string
		metaTitle, metaDescription    *string
		labels                        []string
		content                       json.RawMessage
		imagesRaw                     []byte
		isFeatured, isPublished       bool
		orderIndex                    int
		createdAt, updatedAt          time.Time
		deletedAt                     *time.Time
	)

	if err := row.Scan(
		&id, &title, &slug, &groupID, &categoryID,
		&originalPrice, &discountPercent, &priceDisplayText, &labels,
		&description, &content, &imagesRaw, &metaTitle, &metaDescription,
		&isFeatured, &isPublished, &orderIndex,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	images, err := media.UnmarshalImages(imagesRaw)
	if err != nil {
		return nil, fmt.Errorf("scan service (unmarshal images): %w", err)
	}

	return domain.RehydrateService(
		id, title, slug, groupID, categoryID,
		originalPrice, discountPercent, priceDisplayText, labels, description, content,
		images, metaTitle, metaDescription, isFeatured, isPublished, orderIndex,
		createdAt, updatedAt, deletedAt,
	), nil
}

func scanServiceWithRelations(row rowScanner) (*domain.ServiceWithRelations, error) {
	var (
		id, title, slug                                          string
		groupID, categoryID                                      *string
		originalPrice                                            *int64
		discountPercent                                          *int
		priceDisplayText, description                            *string
		metaTitle, metaDescription                               *string
		labels                                                   []string
		content                                                  json.RawMessage
		imagesRaw                                                []byte
		isFeatured, isPublished                                  bool
		orderIndex                                               int
		createdAt, updatedAt                                     time.Time
		deletedAt                                                *time.Time
		groupRefID, groupRefName, categoryRefID, categoryRefName *string
	)

	if err := row.Scan(
		&id, &title, &slug, &groupID, &categoryID,
		&originalPrice, &discountPercent, &priceDisplayText, &labels,
		&description, &content, &imagesRaw, &metaTitle, &metaDescription,
		&isFeatured, &isPublished, &orderIndex,
		&createdAt, &updatedAt, &deletedAt,
		&groupRefID, &groupRefName, &categoryRefID, &categoryRefName,
	); err != nil {
		return nil, err
	}

	images, err := media.UnmarshalImages(imagesRaw)
	if err != nil {
		return nil, fmt.Errorf("scan service with relations (unmarshal images): %w", err)
	}

	service := domain.RehydrateService(
		id, title, slug, groupID, categoryID,
		originalPrice, discountPercent, priceDisplayText, labels, description, content,
		images, metaTitle, metaDescription, isFeatured, isPublished, orderIndex,
		createdAt, updatedAt, deletedAt,
	)

	result := &domain.ServiceWithRelations{Service: service}
	if groupRefID != nil {
		result.Group = &domain.GroupRef{ID: *groupRefID, Name: *groupRefName}
	}
	if categoryRefID != nil {
		result.Category = &domain.CategoryRef{ID: *categoryRefID, Name: *categoryRefName}
	}

	return result, nil
}
