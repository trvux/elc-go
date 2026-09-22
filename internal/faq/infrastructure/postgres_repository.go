package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/faq/domain"
)

type PostgresFAQRepository struct {
	pool *pgxpool.Pool
}

var _ domain.FAQRepository = (*PostgresFAQRepository)(nil)

func NewPostgresFAQRepository(pool *pgxpool.Pool) *PostgresFAQRepository {
	return &PostgresFAQRepository{pool: pool}
}

const faqColumns = `id, owner_type, owner_id, question, answer, order_index, is_published, created_at, updated_at`

func (r *PostgresFAQRepository) GetByOwner(ctx context.Context, filter domain.FAQFilter) ([]*domain.FAQ, error) {
	query := "SELECT " + faqColumns + " FROM faqs WHERE owner_type = $1 AND owner_id = $2"
	args := []any{string(filter.OwnerType), filter.OwnerID}
	if filter.PublishedOnly {
		query += " AND is_published = true"
	}
	query += " ORDER BY order_index ASC, created_at ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("faq repository getByOwner: %w", err)
	}
	defer rows.Close()

	var faqs []*domain.FAQ
	for rows.Next() {
		faq, err := scanFAQ(rows)
		if err != nil {
			return nil, fmt.Errorf("faq repository getByOwner scan: %w", err)
		}
		faqs = append(faqs, faq)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("faq repository getByOwner rows: %w", err)
	}

	return faqs, nil
}

func (r *PostgresFAQRepository) GetByID(ctx context.Context, id string) (*domain.FAQ, error) {
	query := "SELECT " + faqColumns + " FROM faqs WHERE id = $1"

	row := r.pool.QueryRow(ctx, query, id)
	faq, err := scanFAQ(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("faq repository getById: %w", err)
	}
	return faq, nil
}

func (r *PostgresFAQRepository) Create(ctx context.Context, faq *domain.FAQ) (*domain.FAQ, error) {
	query := `
		INSERT INTO faqs (owner_type, owner_id, question, answer, order_index, is_published)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + faqColumns

	row := r.pool.QueryRow(ctx, query,
		string(faq.OwnerType()), faq.OwnerID(), faq.Question(), faq.Answer(),
		faq.OrderIndex(), faq.IsPublished(),
	)
	created, err := scanFAQ(row)
	if err != nil {
		return nil, fmt.Errorf("faq repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresFAQRepository) Update(ctx context.Context, faq *domain.FAQ) (*domain.FAQ, error) {
	query := `
		UPDATE faqs
		SET question = $1, answer = $2, order_index = $3, is_published = $4, updated_at = $5
		WHERE id = $6
		RETURNING ` + faqColumns

	row := r.pool.QueryRow(ctx, query,
		faq.Question(), faq.Answer(), faq.OrderIndex(), faq.IsPublished(),
		time.Now(), faq.ID(),
	)
	updated, err := scanFAQ(row)
	if err != nil {
		return nil, fmt.Errorf("faq repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresFAQRepository) Delete(ctx context.Context, id string) error {
	if _, err := r.pool.Exec(ctx, "DELETE FROM faqs WHERE id = $1", id); err != nil {
		return fmt.Errorf("faq repository delete: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFAQ(row rowScanner) (*domain.FAQ, error) {
	var (
		id, ownerType, ownerID, question, answer string
		orderIndex                               int
		isPublished                              bool
		createdAt, updatedAt                     time.Time
	)

	if err := row.Scan(
		&id, &ownerType, &ownerID, &question, &answer,
		&orderIndex, &isPublished, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateFAQ(
		id, domain.OwnerType(ownerType), ownerID, question, answer,
		orderIndex, isPublished, createdAt, updatedAt,
	), nil
}
