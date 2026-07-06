package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/event/domain"
)

type PostgresEventRepository struct {
	pool *pgxpool.Pool
}

var _ domain.EventRepository = (*PostgresEventRepository)(nil)

func NewPostgresEventRepository(pool *pgxpool.Pool) *PostgresEventRepository {
	return &PostgresEventRepository{pool: pool}
}

func (r *PostgresEventRepository) Create(ctx context.Context, event *domain.Event) error {
	var entityType *string
	if event.EntityType() != nil {
		s := string(*event.EntityType())
		entityType = &s
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO tracking_events (event_name, event_category, event_label, page_path, session_id)
		VALUES ($1, $2, $3, $4, $5)
	`, string(event.Name()), entityType, event.EntityID(), event.PagePath(), event.SessionID())
	if err != nil {
		return fmt.Errorf("event repository create: %w", err)
	}
	return nil
}

func (r *PostgresEventRepository) TopViewed(ctx context.Context, filter domain.TopViewedFilter) ([]domain.EntityViewCount, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}

	rows, err := r.pool.Query(ctx, `
		SELECT event_label, COUNT(*) AS view_count
		FROM tracking_events
		WHERE event_name = $1 AND event_category = $2 AND created_at >= $3 AND event_label IS NOT NULL
		GROUP BY event_label
		ORDER BY view_count DESC
		LIMIT $4
	`, string(domain.EventViewItem), string(filter.EntityType), filter.Since, limit)
	if err != nil {
		return nil, fmt.Errorf("event repository topViewed: %w", err)
	}
	defer rows.Close()

	var result []domain.EntityViewCount
	for rows.Next() {
		var row domain.EntityViewCount
		if err := rows.Scan(&row.EntityID, &row.Count); err != nil {
			return nil, fmt.Errorf("event repository topViewed scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
