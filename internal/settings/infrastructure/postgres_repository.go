package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/settings/domain"
)

type PostgresSettingsRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSettingsRepository(pool *pgxpool.Pool) *PostgresSettingsRepository {
	return &PostgresSettingsRepository{pool: pool}
}

func (r *PostgresSettingsRepository) GetAll(ctx context.Context) ([]*domain.SiteSetting, error) {
	query := `SELECT key, value FROM site_settings`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("settings repository getAll: %w", err)
	}
	defer rows.Close()

	var settings []*domain.SiteSetting
	for rows.Next() {
		var key string
		var value *string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("settings repository getAll scan: %w", err)
		}
		valStr := ""
		if value != nil {
			valStr = *value
		}
		settings = append(settings, domain.RehydrateSiteSetting(key, valStr))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("settings repository getAll rows: %w", err)
	}

	return settings, nil
}

func (r *PostgresSettingsRepository) UpdateMany(ctx context.Context, settings []*domain.SiteSetting) error {
	if len(settings) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("settings repository updateMany (begin tx): %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	query := `
		INSERT INTO site_settings (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`

	for _, s := range settings {
		_, err := tx.Exec(ctx, query, s.Key(), s.Value())
		if err != nil {
			return fmt.Errorf("settings repository updateMany upsert %s: %w", s.Key(), err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("settings repository updateMany (commit): %w", err)
	}

	return nil
}
