package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/slug-registry/domain"
)

// PostgresSlugRegistryRepository reads the shared slug_registry table —
// every entity module (products, projects, categories, brands, groups,
// project_type) maintains its own rows there via DB triggers to enforce
// slug uniqueness across all entity types; this repository never writes.
type PostgresSlugRegistryRepository struct {
	pool *pgxpool.Pool
}

var _ domain.SlugRegistryRepository = (*PostgresSlugRegistryRepository)(nil)

func NewPostgresSlugRegistryRepository(pool *pgxpool.Pool) *PostgresSlugRegistryRepository {
	return &PostgresSlugRegistryRepository{pool: pool}
}

func (r *PostgresSlugRegistryRepository) GetBySlug(ctx context.Context, slug string) (*domain.SlugRegistryEntry, error) {
	query := `SELECT entity_type, entity_id FROM slug_registry WHERE slug = $1 AND deleted_at IS NULL`

	var entityType, entityID string
	err := r.pool.QueryRow(ctx, query, slug).Scan(&entityType, &entityID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("slug registry repository getBySlug: %w", err)
	}

	return domain.RehydrateSlugRegistryEntry(entityType, entityID), nil
}
