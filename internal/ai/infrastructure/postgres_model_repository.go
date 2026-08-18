package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/ai/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type PostgresModelRepository struct {
	pool   *pgxpool.Pool
	cipher *SecretCipher
}

func NewPostgresModelRepository(pool *pgxpool.Pool, cipher *SecretCipher) *PostgresModelRepository {
	return &PostgresModelRepository{pool: pool, cipher: cipher}
}

var _ domain.ModelRepository = (*PostgresModelRepository)(nil)

const modelColumns = "id, provider_id, model_name, display_name, role, pricing, fallback_priority, is_default, is_active, created_at, updated_at"

func scanModel(row pgx.Row) (*domain.Model, error) {
	var (
		id, providerID, modelName, displayName, role string
		pricingRaw                                   []byte
		fallbackPriority                             int
		isDefault, isActive                          bool
		createdAt, updatedAt                         time.Time
	)
	if err := row.Scan(&id, &providerID, &modelName, &displayName, &role, &pricingRaw, &fallbackPriority, &isDefault, &isActive, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var pricing domain.Pricing
	if err := json.Unmarshal(pricingRaw, &pricing); err != nil {
		return nil, fmt.Errorf("unmarshal pricing: %w", err)
	}
	return domain.RehydrateModel(id, providerID, modelName, displayName, domain.ModelRole(role), pricing, fallbackPriority, isDefault, isActive, createdAt, updatedAt), nil
}

func (r *PostgresModelRepository) Create(ctx context.Context, m *domain.Model) (*domain.Model, error) {
	pricingJSON, err := json.Marshal(m.Pricing())
	if err != nil {
		return nil, fmt.Errorf("ai model repository create (marshal pricing): %w", err)
	}

	query := `
		INSERT INTO ai_models (provider_id, model_name, display_name, role, pricing, fallback_priority, is_default, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ` + modelColumns

	row := r.pool.QueryRow(ctx, query,
		m.ProviderID(), m.ModelName(), m.DisplayName(), string(m.Role()), pricingJSON, m.FallbackPriority(), m.IsDefault(), m.IsActive(),
	)
	created, err := scanModel(row)
	if err != nil {
		return nil, fmt.Errorf("ai model repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresModelRepository) Update(ctx context.Context, m *domain.Model) (*domain.Model, error) {
	pricingJSON, err := json.Marshal(m.Pricing())
	if err != nil {
		return nil, fmt.Errorf("ai model repository update (marshal pricing): %w", err)
	}

	query := `
		UPDATE ai_models
		SET display_name = $1, pricing = $2, fallback_priority = $3, is_default = $4, is_active = $5, updated_at = $6
		WHERE id = $7
		RETURNING ` + modelColumns

	row := r.pool.QueryRow(ctx, query, m.DisplayName(), pricingJSON, m.FallbackPriority(), m.IsDefault(), m.IsActive(), m.UpdatedAt(), m.ID())
	updated, err := scanModel(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NewNotFoundError("ai model")
		}
		return nil, fmt.Errorf("ai model repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresModelRepository) List(ctx context.Context) ([]*domain.Model, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+modelColumns+" FROM ai_models ORDER BY role, fallback_priority")
	if err != nil {
		return nil, fmt.Errorf("ai model repository list: %w", err)
	}
	defer rows.Close()

	var models []*domain.Model
	for rows.Next() {
		m, err := scanModel(rows)
		if err != nil {
			return nil, fmt.Errorf("ai model repository list (scan): %w", err)
		}
		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ai model repository list (rows): %w", err)
	}
	return models, nil
}

func (r *PostgresModelRepository) GetByID(ctx context.Context, id string) (*domain.Model, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+modelColumns+" FROM ai_models WHERE id = $1", id)
	m, err := scanModel(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ai model repository getByID: %w", err)
	}
	return m, nil
}

func (r *PostgresModelRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM ai_models WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("ai model repository delete: %w", err)
	}
	return nil
}

// ListActiveConfigsByRole is the one read SendChatMessage's fallback loop
// actually drives off: active models of role, joined with their (active)
// provider's base URL and decrypted API key, ordered by fallback_priority.
func (r *PostgresModelRepository) ListActiveConfigsByRole(ctx context.Context, role domain.ModelRole) ([]domain.ModelConfig, error) {
	query := `
		SELECT m.id, m.model_name, m.pricing, m.fallback_priority,
		       p.id, p.name, p.base_url, p.api_key_encrypted
		FROM ai_models m
		JOIN ai_providers p ON p.id = m.provider_id
		WHERE m.role = $1 AND m.is_active AND p.is_active
		ORDER BY m.fallback_priority ASC`

	rows, err := r.pool.Query(ctx, query, string(role))
	if err != nil {
		return nil, fmt.Errorf("ai model repository listActiveConfigsByRole: %w", err)
	}
	defer rows.Close()

	var configs []domain.ModelConfig
	for rows.Next() {
		var (
			modelID, modelName                string
			pricingRaw                        []byte
			fallbackPriority                  int
			providerID, providerName, baseURL string
			apiKeyEncrypted                   []byte
		)
		if err := rows.Scan(&modelID, &modelName, &pricingRaw, &fallbackPriority, &providerID, &providerName, &baseURL, &apiKeyEncrypted); err != nil {
			return nil, fmt.Errorf("ai model repository listActiveConfigsByRole (scan): %w", err)
		}
		var pricing domain.Pricing
		if err := json.Unmarshal(pricingRaw, &pricing); err != nil {
			return nil, fmt.Errorf("ai model repository listActiveConfigsByRole (unmarshal pricing): %w", err)
		}
		apiKey, err := r.cipher.Decrypt(apiKeyEncrypted)
		if err != nil {
			return nil, fmt.Errorf("ai model repository listActiveConfigsByRole (decrypt api key for provider %s): %w", providerName, err)
		}
		configs = append(configs, domain.ModelConfig{
			ProviderID: providerID, ProviderName: providerName, BaseURL: baseURL, APIKey: apiKey,
			ModelID: modelID, ModelName: modelName, Role: role, Pricing: pricing, FallbackPriority: fallbackPriority,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ai model repository listActiveConfigsByRole (rows): %w", err)
	}
	return configs, nil
}

// UpdatePricing is used only by cmd/sync-ai-pricing — see ModelRepository's
// doc comment: it never creates a row, only updates an existing
// (providerID, modelName) match.
func (r *PostgresModelRepository) UpdatePricing(ctx context.Context, providerID, modelName string, pricing domain.Pricing) (bool, error) {
	pricingJSON, err := json.Marshal(pricing)
	if err != nil {
		return false, fmt.Errorf("ai model repository updatePricing (marshal): %w", err)
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE ai_models SET pricing = $1, updated_at = now() WHERE provider_id = $2 AND model_name = $3`,
		pricingJSON, providerID, modelName,
	)
	if err != nil {
		return false, fmt.Errorf("ai model repository updatePricing: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
