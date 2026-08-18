package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/ai/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type PostgresProviderRepository struct {
	pool   *pgxpool.Pool
	cipher *SecretCipher
}

func NewPostgresProviderRepository(pool *pgxpool.Pool, cipher *SecretCipher) *PostgresProviderRepository {
	return &PostgresProviderRepository{pool: pool, cipher: cipher}
}

var _ domain.ProviderRepository = (*PostgresProviderRepository)(nil)

const providerColumns = "id, name, display_name, base_url, pricing_doc_url, is_active, created_at, updated_at"

// scanProvider reads providerColumns — never api_key_encrypted, so a
// *domain.Provider coming out of this repository always has APIKey() == ""
// (see Provider's and ProviderRepository's doc comments: the plaintext key
// is write-only, never round-tripped back out).
func scanProvider(row pgx.Row) (*domain.Provider, error) {
	var (
		id, name, displayName, baseURL string
		pricingDocURL                  *string
		isActive                       bool
		createdAt, updatedAt           time.Time
	)
	if err := row.Scan(&id, &name, &displayName, &baseURL, &pricingDocURL, &isActive, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	docURL := ""
	if pricingDocURL != nil {
		docURL = *pricingDocURL
	}
	return domain.RehydrateProvider(id, name, displayName, baseURL, "", docURL, isActive, createdAt, updatedAt), nil
}

func (r *PostgresProviderRepository) Create(ctx context.Context, p *domain.Provider) (*domain.Provider, error) {
	encKey, err := r.cipher.Encrypt(p.APIKey())
	if err != nil {
		return nil, fmt.Errorf("ai provider repository create (encrypt api key): %w", err)
	}

	query := `
		INSERT INTO ai_providers (name, display_name, base_url, pricing_doc_url, api_key_encrypted, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + providerColumns

	row := r.pool.QueryRow(ctx, query,
		p.Name(), p.DisplayName(), p.BaseURL(), nullIfEmpty(p.PricingDocURL()), encKey, p.IsActive(),
	)
	created, err := scanProvider(row)
	if err != nil {
		return nil, fmt.Errorf("ai provider repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresProviderRepository) Update(ctx context.Context, p *domain.Provider, newAPIKey *string) (*domain.Provider, error) {
	var row pgx.Row
	if newAPIKey != nil {
		encKey, err := r.cipher.Encrypt(*newAPIKey)
		if err != nil {
			return nil, fmt.Errorf("ai provider repository update (encrypt api key): %w", err)
		}
		query := `
			UPDATE ai_providers
			SET display_name = $1, base_url = $2, pricing_doc_url = $3, api_key_encrypted = $4, is_active = $5, updated_at = $6
			WHERE id = $7
			RETURNING ` + providerColumns
		row = r.pool.QueryRow(ctx, query, p.DisplayName(), p.BaseURL(), nullIfEmpty(p.PricingDocURL()), encKey, p.IsActive(), p.UpdatedAt(), p.ID())
	} else {
		query := `
			UPDATE ai_providers
			SET display_name = $1, base_url = $2, pricing_doc_url = $3, is_active = $4, updated_at = $5
			WHERE id = $6
			RETURNING ` + providerColumns
		row = r.pool.QueryRow(ctx, query, p.DisplayName(), p.BaseURL(), nullIfEmpty(p.PricingDocURL()), p.IsActive(), p.UpdatedAt(), p.ID())
	}

	updated, err := scanProvider(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NewNotFoundError("ai provider")
		}
		return nil, fmt.Errorf("ai provider repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresProviderRepository) List(ctx context.Context) ([]*domain.Provider, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+providerColumns+" FROM ai_providers ORDER BY created_at")
	if err != nil {
		return nil, fmt.Errorf("ai provider repository list: %w", err)
	}
	defer rows.Close()

	var providers []*domain.Provider
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, fmt.Errorf("ai provider repository list (scan): %w", err)
		}
		providers = append(providers, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ai provider repository list (rows): %w", err)
	}
	return providers, nil
}

func (r *PostgresProviderRepository) GetByID(ctx context.Context, id string) (*domain.Provider, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+providerColumns+" FROM ai_providers WHERE id = $1", id)
	p, err := scanProvider(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ai provider repository getByID: %w", err)
	}
	return p, nil
}

func (r *PostgresProviderRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM ai_providers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("ai provider repository delete: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
