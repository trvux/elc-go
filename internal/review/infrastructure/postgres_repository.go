package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/review/domain"
)

type PostgresReviewRepository struct {
	pool *pgxpool.Pool
}

var _ domain.ReviewRepository = (*PostgresReviewRepository)(nil)

func NewPostgresReviewRepository(pool *pgxpool.Pool) *PostgresReviewRepository {
	return &PostgresReviewRepository{pool: pool}
}

const reviewColumns = `id, product_id, service_id, rating, comment, reviewer_name, reviewer_phone,
	is_published, source_ip, user_agent, created_at, updated_at`

func (r *PostgresReviewRepository) Create(ctx context.Context, review *domain.Review) (*domain.Review, error) {
	query := `
		INSERT INTO reviews (product_id, service_id, rating, comment, reviewer_name, reviewer_phone, is_published, source_ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + reviewColumns

	row := r.pool.QueryRow(ctx, query,
		review.ProductID(), review.ServiceID(), review.Rating(), review.Comment(),
		review.ReviewerName(), review.ReviewerPhone(), review.IsPublished(),
		review.SourceIP(), review.UserAgent(),
	)
	created, err := scanReview(row)
	if err != nil {
		return nil, fmt.Errorf("review repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresReviewRepository) GetAll(ctx context.Context, filter domain.ReviewFilter) ([]*domain.Review, error) {
	query := "SELECT " + reviewColumns + " FROM reviews"
	conditions, args := reviewFilterConditions(filter)
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"

	argN := len(args) + 1
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

	var reviews []*domain.Review
	for rows.Next() {
		review, err := scanReview(rows)
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
	query := "SELECT COUNT(*) FROM reviews"
	conditions, args := reviewFilterConditions(filter)
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("review repository count: %w", err)
	}
	return count, nil
}

func (r *PostgresReviewRepository) GetByID(ctx context.Context, id string) (*domain.Review, error) {
	query := "SELECT " + reviewColumns + " FROM reviews WHERE id = $1"

	row := r.pool.QueryRow(ctx, query, id)
	review, err := scanReview(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("review repository getById: %w", err)
	}
	return review, nil
}

func (r *PostgresReviewRepository) Update(ctx context.Context, review *domain.Review) (*domain.Review, error) {
	query := `
		UPDATE reviews
		SET is_published = $1, updated_at = now()
		WHERE id = $2
		RETURNING ` + reviewColumns

	row := r.pool.QueryRow(ctx, query, review.IsPublished(), review.ID())
	updated, err := scanReview(row)
	if err != nil {
		return nil, fmt.Errorf("review repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresReviewRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM reviews WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("review repository delete: %w", err)
	}
	return nil
}

func (r *PostgresReviewRepository) GetSummary(ctx context.Context, productID, serviceID *string) (*domain.ReviewSummary, error) {
	query := `
		SELECT COUNT(*), COALESCE(AVG(rating), 0)
		FROM reviews
		WHERE is_published = true`
	args := []any{}
	argN := 1

	if productID != nil && *productID != "" {
		query += fmt.Sprintf(" AND product_id = $%d", argN)
		args = append(args, *productID)
		argN++
	}
	if serviceID != nil && *serviceID != "" {
		query += fmt.Sprintf(" AND service_id = $%d", argN)
		args = append(args, *serviceID)
		argN++
	}

	var summary domain.ReviewSummary
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&summary.Count, &summary.Average); err != nil {
		return nil, fmt.Errorf("review repository getSummary: %w", err)
	}
	return &summary, nil
}

// reviewFilterConditions builds WHERE clauses shared by GetAll/Count so the
// two queries can never drift out of sync with each other.
func reviewFilterConditions(filter domain.ReviewFilter) ([]string, []any) {
	conditions := []string{}
	args := []any{}
	argN := 1

	if filter.ProductID != nil && *filter.ProductID != "" {
		conditions = append(conditions, fmt.Sprintf("product_id = $%d", argN))
		args = append(args, *filter.ProductID)
		argN++
	}
	if filter.ServiceID != nil && *filter.ServiceID != "" {
		conditions = append(conditions, fmt.Sprintf("service_id = $%d", argN))
		args = append(args, *filter.ServiceID)
		argN++
	}
	if filter.IsPublished != nil {
		conditions = append(conditions, fmt.Sprintf("is_published = $%d", argN))
		args = append(args, *filter.IsPublished)
		argN++
	}

	return conditions, args
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanReview(row rowScanner) (*domain.Review, error) {
	var (
		id, comment, reviewerName string
		productID, serviceID      *string
		rating                    int
		reviewerPhone             *string
		isPublished               bool
		sourceIP                  *string
		userAgent                 *string
		createdAt, updatedAt      time.Time
	)

	if err := row.Scan(
		&id, &productID, &serviceID, &rating, &comment, &reviewerName, &reviewerPhone,
		&isPublished, &sourceIP, &userAgent,
		&createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateReview(
		id, productID, serviceID, rating, comment, reviewerName, reviewerPhone,
		isPublished, sourceIP, userAgent,
		createdAt, updatedAt,
	), nil
}
