package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/review/domain"
)

// PostgresReviewRepository implements domain.ReviewRepository against the
// reviews table.
type PostgresReviewRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ReviewRepository = (*PostgresReviewRepository)(nil)

func NewPostgresReviewRepository(pool *pgxpool.Pool) *PostgresReviewRepository {
	return &PostgresReviewRepository{pool: pool}
}

const reviewColumns = `id, product_id, project_id, service_id, news_id, rating, comment,
	reviewer_name, reviewer_phone, is_published, source_ip, user_agent, created_at, updated_at`

// reviewColumnsAliased (with an "r" alias) + reviewProductJoin back GetAll/
// Count, which LEFT JOIN products so a product review's name/slug comes
// back alongside it — same pattern internal/category uses for its own
// group join, see internal/category/infrastructure/postgres_repository.go.
const reviewColumnsAliased = `r.id, r.product_id, r.project_id, r.service_id, r.news_id, r.rating, r.comment,
	r.reviewer_name, r.reviewer_phone, r.is_published, r.source_ip, r.user_agent, r.created_at, r.updated_at`

const reviewProductJoin = `SELECT ` + reviewColumnsAliased + `, p.id, p.name, p.slug
	FROM reviews r LEFT JOIN products p ON p.id = r.product_id`

func (r *PostgresReviewRepository) Create(ctx context.Context, review *domain.Review) (*domain.Review, error) {
	query := `
		INSERT INTO reviews (product_id, project_id, service_id, news_id, rating, comment, reviewer_name, reviewer_phone, is_published, source_ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING ` + reviewColumns

	row := r.pool.QueryRow(ctx, query,
		review.ProductID(), review.ProjectID(), review.ServiceID(), review.NewsID(),
		review.Rating(), review.Comment(), review.ReviewerName(), review.ReviewerPhone(),
		review.IsPublished(), review.SourceIP(), review.UserAgent(),
	)
	created, err := scanReview(row)
	if err != nil {
		return nil, fmt.Errorf("review repository create: %w", err)
	}
	return created, nil
}

// GetByEntity — entityColumn is caller-controlled from a fixed whitelist
// (see presentation.entityColumnFor), never raw request input, so building
// the WHERE clause with it directly (not as a bind parameter — Postgres
// can't parameterize identifiers) is safe.
func (r *PostgresReviewRepository) GetByEntity(ctx context.Context, entityColumn, entityID string) ([]*domain.Review, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM reviews WHERE %s = $1 AND is_published = true ORDER BY created_at DESC",
		reviewColumns, entityColumn,
	)

	rows, err := r.pool.Query(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("review repository getByEntity: %w", err)
	}
	defer rows.Close()

	var reviews []*domain.Review
	for rows.Next() {
		review, err := scanReview(rows)
		if err != nil {
			return nil, fmt.Errorf("review repository getByEntity scan: %w", err)
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("review repository getByEntity rows: %w", err)
	}

	return reviews, nil
}

// GetAll backs the staff-only admin list — every review regardless of
// entity type or is_published, newest first.
func (r *PostgresReviewRepository) GetAll(ctx context.Context, filter domain.ReviewFilter) ([]*domain.ReviewWithProduct, error) {
	query := reviewProductJoin + " ORDER BY r.created_at DESC"

	var args []any
	argN := 1
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, filter.Limit)
		argN++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argN)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("review repository getAll: %w", err)
	}
	defer rows.Close()

	var reviews []*domain.ReviewWithProduct
	for rows.Next() {
		review, err := scanReviewWithProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("review repository getAll scan: %w", err)
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("review repository getAll rows: %w", err)
	}

	return reviews, nil
}

func (r *PostgresReviewRepository) Count(ctx context.Context, filter domain.ReviewFilter) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM reviews").Scan(&count); err != nil {
		return 0, fmt.Errorf("review repository count: %w", err)
	}
	return count, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanReview(row rowScanner) (*domain.Review, error) {
	var (
		id                                      string
		productID, projectID, serviceID, newsID *string
		rating                                  int
		comment, reviewerName                   string
		reviewerPhone                           *string
		isPublished                             bool
		sourceIP, userAgent                     *string
		createdAt, updatedAt                    time.Time
	)

	if err := row.Scan(
		&id, &productID, &projectID, &serviceID, &newsID,
		&rating, &comment, &reviewerName, &reviewerPhone,
		&isPublished, &sourceIP, &userAgent,
		&createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateReview(
		id, productID, projectID, serviceID, newsID,
		rating, comment, reviewerName, reviewerPhone,
		isPublished, sourceIP, userAgent,
		createdAt, updatedAt,
	), nil
}

// scanReviewWithProduct scans one row of reviewProductJoin — the trailing
// p.id/p.name/p.slug come back NULL when the review has no product_id or
// its product was hard-deleted (LEFT JOIN), even though those columns are
// NOT NULL on products itself.
func scanReviewWithProduct(row rowScanner) (*domain.ReviewWithProduct, error) {
	var (
		id                                           string
		productID, projectID, serviceID, newsID      *string
		rating                                       int
		comment, reviewerName                        string
		reviewerPhone                                *string
		isPublished                                  bool
		sourceIP, userAgent                          *string
		createdAt, updatedAt                         time.Time
		refProductID, refProductName, refProductSlug *string
	)

	if err := row.Scan(
		&id, &productID, &projectID, &serviceID, &newsID,
		&rating, &comment, &reviewerName, &reviewerPhone,
		&isPublished, &sourceIP, &userAgent,
		&createdAt, &updatedAt,
		&refProductID, &refProductName, &refProductSlug,
	); err != nil {
		return nil, err
	}

	review := domain.RehydrateReview(
		id, productID, projectID, serviceID, newsID,
		rating, comment, reviewerName, reviewerPhone,
		isPublished, sourceIP, userAgent,
		createdAt, updatedAt,
	)

	var product *domain.ProductRef
	if refProductID != nil {
		product = &domain.ProductRef{ID: *refProductID, Name: *refProductName, Slug: *refProductSlug}
	}

	return &domain.ReviewWithProduct{Review: review, Product: product}, nil
}
