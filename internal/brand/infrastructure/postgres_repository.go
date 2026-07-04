package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/brand/domain"
)

type PostgresBrandRepository struct {
	pool *pgxpool.Pool
}

var _ domain.BrandRepository = (*PostgresBrandRepository)(nil)

func NewPostgresBrandRepository(pool *pgxpool.Pool) *PostgresBrandRepository {
	return &PostgresBrandRepository{pool: pool}
}

const brandColumns = `id, name, slug, logo_url, meta_title, meta_description,
	is_featured, order_index, content, faq, created_at, updated_at, deleted_at`

func (r *PostgresBrandRepository) GetAll(ctx context.Context, filter domain.BrandFilter) ([]*domain.Brand, error) {
	query := "SELECT " + brandColumns + " FROM brands"
	conditions := []string{}
	args := []any{}
	argN := 1

	if !filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argN))
		args = append(args, "%"+filter.Search+"%")
		argN++
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY order_index ASC, name ASC"

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
		return nil, fmt.Errorf("brand repository getAll: %w", err)
	}
	defer rows.Close()

	var brands []*domain.Brand
	for rows.Next() {
		b, err := scanBrand(rows)
		if err != nil {
			return nil, fmt.Errorf("brand repository getAll scan: %w", err)
		}
		brands = append(brands, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("brand repository getAll rows: %w", err)
	}

	return brands, nil
}

func (r *PostgresBrandRepository) GetByID(ctx context.Context, id string) (*domain.Brand, error) {
	query := "SELECT " + brandColumns + " FROM brands WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	b, err := scanBrand(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("brand repository getById: %w", err)
	}
	return b, nil
}

func (r *PostgresBrandRepository) GetBySlug(ctx context.Context, slug string) (*domain.Brand, error) {
	query := "SELECT " + brandColumns + " FROM brands WHERE slug = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, slug)
	b, err := scanBrand(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("brand repository getBySlug: %w", err)
	}
	return b, nil
}

// Create is a plain INSERT — unlike service-group, brands.slug is only
// unique among non-deleted rows (a partial index), so a soft-deleted brand
// never blocks reusing its slug and no "resurrect" step is needed here.
// brands.name, however, IS a full unique constraint (applies even to
// soft-deleted rows) — see docs/brand.md for the resulting gotcha, carried
// over unchanged from the old TS behavior.
func (r *PostgresBrandRepository) Create(ctx context.Context, brand *domain.Brand) (*domain.Brand, error) {
	query := `
		INSERT INTO brands (name, slug, logo_url, meta_title, meta_description, is_featured, order_index, content, faq)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + brandColumns

	faqJSON, err := marshalFAQ(brand.FAQ())
	if err != nil {
		return nil, fmt.Errorf("brand repository create (marshal faq): %w", err)
	}

	row := r.pool.QueryRow(ctx, query,
		brand.Name(), brand.Slug(), brand.LogoURL(), brand.MetaTitle(), brand.MetaDescription(),
		brand.IsFeatured(), brand.OrderIndex(), brand.Content(), faqJSON,
	)
	created, err := scanBrand(row)
	if err != nil {
		return nil, fmt.Errorf("brand repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresBrandRepository) Update(ctx context.Context, brand *domain.Brand) (*domain.Brand, error) {
	query := `
		UPDATE brands
		SET name = $1, slug = $2, logo_url = $3, meta_title = $4, meta_description = $5,
			is_featured = $6, order_index = $7, content = $8, faq = $9, updated_at = $10
		WHERE id = $11
		RETURNING ` + brandColumns

	faqJSON, err := marshalFAQ(brand.FAQ())
	if err != nil {
		return nil, fmt.Errorf("brand repository update (marshal faq): %w", err)
	}

	row := r.pool.QueryRow(ctx, query,
		brand.Name(), brand.Slug(), brand.LogoURL(), brand.MetaTitle(), brand.MetaDescription(),
		brand.IsFeatured(), brand.OrderIndex(), brand.Content(), faqJSON,
		brand.UpdatedAt(), brand.ID(),
	)
	updated, err := scanBrand(row)
	if err != nil {
		return nil, fmt.Errorf("brand repository update: %w", err)
	}
	return updated, nil
}

// SoftDelete only ever touches the brands row itself. Unlike service-group's
// services.group_id (nullable), products.brand_id is NOT NULL — the FK's
// "ON DELETE SET NULL" is actually unreachable on this schema (a real hard
// DELETE on a referenced brand would itself fail the NOT NULL check first).
// So there is nothing to null out here: any product rows keep pointing at
// the now-soft-deleted brand row, exactly like the old TS `delete()` did.
// See docs/brand.md.
func (r *PostgresBrandRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE brands SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("brand repository softDelete: %w", err)
	}
	return nil
}

func (r *PostgresBrandRepository) Restore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE brands SET deleted_at = NULL, updated_at = $1 WHERE id = $2",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("brand repository restore: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

// marshalFAQ/unmarshalFAQ hand-roll the jsonb <-> []domain.FAQItem
// conversion because the shared pool runs in pgx's simple protocol mode
// (see internal/platform/db — required to avoid the PgBouncer prepared-
// statement gotcha, docs/contact.md). Simple protocol can't infer an OID for
// an arbitrary struct slice; the return type must be json.RawMessage
// specifically (not a plain []byte) — pgx encodes json.RawMessage as raw
// JSON text but a bare []byte as a bytea literal, which Postgres then
// rejects trying to cast to jsonb ("invalid input syntax for type json").
func marshalFAQ(faq []domain.FAQItem) (json.RawMessage, error) {
	if faq == nil {
		return nil, nil
	}
	return json.Marshal(faq)
}

func unmarshalFAQ(raw []byte) ([]domain.FAQItem, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var faq []domain.FAQItem
	if err := json.Unmarshal(raw, &faq); err != nil {
		return nil, err
	}
	return faq, nil
}

func scanBrand(row rowScanner) (*domain.Brand, error) {
	var (
		id, name, slug, logoURL    string
		metaTitle, metaDescription *string
		isFeatured                 bool
		orderIndex                 int
		content                    json.RawMessage
		faqRaw                     []byte
		createdAt, updatedAt       time.Time
		deletedAt                  *time.Time
	)

	if err := row.Scan(
		&id, &name, &slug, &logoURL, &metaTitle, &metaDescription,
		&isFeatured, &orderIndex, &content, &faqRaw, &createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	faq, err := unmarshalFAQ(faqRaw)
	if err != nil {
		return nil, fmt.Errorf("scan brand (unmarshal faq): %w", err)
	}

	return domain.RehydrateBrand(
		id, name, slug, logoURL, metaTitle, metaDescription,
		isFeatured, orderIndex, content, faq, createdAt, updatedAt, deletedAt,
	), nil
}
