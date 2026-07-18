package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/recently-viewed/domain"
)

type PostgresRecentlyViewedRepository struct {
	pool *pgxpool.Pool
}

var _ domain.RecentlyViewedRepository = (*PostgresRecentlyViewedRepository)(nil)

func NewPostgresRecentlyViewedRepository(pool *pgxpool.Pool) *PostgresRecentlyViewedRepository {
	return &PostgresRecentlyViewedRepository{pool: pool}
}

const recentlyViewedItemColumns = `id, visitor_id, product_id, viewed_at`

type rowScanner interface {
	Scan(dest ...any) error
}

// Record upserts on the (visitor_id, product_id) unique constraint — viewing
// a product already on the list bumps its viewed_at instead of creating a
// second row, so List's ORDER BY viewed_at DESC always reflects each
// product's most recent view, not its first.
func (r *PostgresRecentlyViewedRepository) Record(ctx context.Context, item *domain.RecentlyViewedItem) (*domain.RecentlyViewedItem, error) {
	query := `
		INSERT INTO recently_viewed_items (visitor_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT (visitor_id, product_id) DO UPDATE SET viewed_at = now()
		RETURNING ` + recentlyViewedItemColumns

	row := r.pool.QueryRow(ctx, query, item.VisitorID(), item.ProductID())
	recorded, err := scanRecentlyViewedItem(row)
	if err != nil {
		return nil, fmt.Errorf("recently-viewed repository record: %w", err)
	}
	return recorded, nil
}

// List caps at 20 rows — see RecentlyViewedRepository's doc comment on why
// there's no separate cleanup job for the underlying table.
func (r *PostgresRecentlyViewedRepository) List(ctx context.Context, visitorID string) ([]*domain.RecentlyViewedItemWithProduct, error) {
	query := `
		SELECT ri.id, ri.visitor_id, ri.product_id, ri.viewed_at,
		       p.id, p.name, p.slug, p.images, p.display_price
		FROM recently_viewed_items ri
		JOIN products p ON p.id = ri.product_id AND p.deleted_at IS NULL
		WHERE ri.visitor_id = $1
		ORDER BY ri.viewed_at DESC
		LIMIT 20`

	rows, err := r.pool.Query(ctx, query, visitorID)
	if err != nil {
		return nil, fmt.Errorf("recently-viewed repository list: %w", err)
	}
	defer rows.Close()

	var items []*domain.RecentlyViewedItemWithProduct
	for rows.Next() {
		var (
			id, itemVisitorID, productID string
			viewedAt                     time.Time
			prodID, prodName, prodSlug   string
			imagesRaw                    []byte
			displayPrice                 *int64
		)
		if err := rows.Scan(
			&id, &itemVisitorID, &productID, &viewedAt,
			&prodID, &prodName, &prodSlug, &imagesRaw, &displayPrice,
		); err != nil {
			return nil, fmt.Errorf("recently-viewed repository list scan: %w", err)
		}

		images, err := media.UnmarshalImages(imagesRaw)
		if err != nil {
			return nil, fmt.Errorf("recently-viewed repository list (unmarshal images): %w", err)
		}

		items = append(items, &domain.RecentlyViewedItemWithProduct{
			RecentlyViewedItem: domain.RehydrateRecentlyViewedItem(id, itemVisitorID, productID, viewedAt),
			Product: &domain.ProductSummary{
				ID: prodID, Name: prodName, Slug: prodSlug,
				ImageURL: media.FirstURL(images), DisplayPrice: displayPrice,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recently-viewed repository list rows: %w", err)
	}
	return items, nil
}

func scanRecentlyViewedItem(row rowScanner) (*domain.RecentlyViewedItem, error) {
	var (
		id, visitorID, productID string
		viewedAt                 time.Time
	)
	if err := row.Scan(&id, &visitorID, &productID, &viewedAt); err != nil {
		return nil, err
	}
	return domain.RehydrateRecentlyViewedItem(id, visitorID, productID, viewedAt), nil
}
