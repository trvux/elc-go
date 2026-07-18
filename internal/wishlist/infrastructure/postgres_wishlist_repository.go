package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/wishlist/domain"
)

type PostgresWishlistRepository struct {
	pool *pgxpool.Pool
}

var _ domain.WishlistRepository = (*PostgresWishlistRepository)(nil)

func NewPostgresWishlistRepository(pool *pgxpool.Pool) *PostgresWishlistRepository {
	return &PostgresWishlistRepository{pool: pool}
}

const wishlistItemColumns = `id, visitor_id, product_id, created_at`

type rowScanner interface {
	Scan(dest ...any) error
}

// Add upserts on the (visitor_id, product_id) unique constraint — the
// DO UPDATE (rather than DO NOTHING) is the standard idiom to still get a row
// back through RETURNING when the item already exists, matching
// WishlistRepository's doc comment: adding an already-wishlisted product is a
// no-op, not an error.
func (r *PostgresWishlistRepository) Add(ctx context.Context, item *domain.WishlistItem) (*domain.WishlistItem, error) {
	query := `
		INSERT INTO wishlist_items (visitor_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT (visitor_id, product_id) DO UPDATE SET visitor_id = EXCLUDED.visitor_id
		RETURNING ` + wishlistItemColumns

	row := r.pool.QueryRow(ctx, query, item.VisitorID(), item.ProductID())
	created, err := scanWishlistItem(row)
	if err != nil {
		return nil, fmt.Errorf("wishlist repository add: %w", err)
	}
	return created, nil
}

func (r *PostgresWishlistRepository) Remove(ctx context.Context, visitorID, productID string) error {
	if _, err := r.pool.Exec(ctx, "DELETE FROM wishlist_items WHERE visitor_id = $1 AND product_id = $2", visitorID, productID); err != nil {
		return fmt.Errorf("wishlist repository remove: %w", err)
	}
	return nil
}

// List joins directly into `products` (owned by the sibling internal/product
// module) for the display summary — same cross-module SQL-join convention
// product itself uses for CategoryRef/BrandRef (see
// internal/product/infrastructure/postgres_repository.go).
func (r *PostgresWishlistRepository) List(ctx context.Context, visitorID string) ([]*domain.WishlistItemWithProduct, error) {
	query := `
		SELECT wi.id, wi.visitor_id, wi.product_id, wi.created_at,
		       p.id, p.name, p.slug, p.images, p.display_price
		FROM wishlist_items wi
		JOIN products p ON p.id = wi.product_id AND p.deleted_at IS NULL
		WHERE wi.visitor_id = $1
		ORDER BY wi.created_at DESC`

	rows, err := r.pool.Query(ctx, query, visitorID)
	if err != nil {
		return nil, fmt.Errorf("wishlist repository list: %w", err)
	}
	defer rows.Close()

	var items []*domain.WishlistItemWithProduct
	for rows.Next() {
		var (
			id, itemVisitorID, productID string
			createdAt                    time.Time
			prodID, prodName, prodSlug   string
			imagesRaw                    []byte
			displayPrice                 *int64
		)
		if err := rows.Scan(
			&id, &itemVisitorID, &productID, &createdAt,
			&prodID, &prodName, &prodSlug, &imagesRaw, &displayPrice,
		); err != nil {
			return nil, fmt.Errorf("wishlist repository list scan: %w", err)
		}

		images, err := media.UnmarshalImages(imagesRaw)
		if err != nil {
			return nil, fmt.Errorf("wishlist repository list (unmarshal images): %w", err)
		}

		items = append(items, &domain.WishlistItemWithProduct{
			WishlistItem: domain.RehydrateWishlistItem(id, itemVisitorID, productID, createdAt),
			Product: &domain.ProductSummary{
				ID: prodID, Name: prodName, Slug: prodSlug,
				ImageURL: media.FirstURL(images), DisplayPrice: displayPrice,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("wishlist repository list rows: %w", err)
	}
	return items, nil
}

func scanWishlistItem(row rowScanner) (*domain.WishlistItem, error) {
	var (
		id, visitorID, productID string
		createdAt                time.Time
	)
	if err := row.Scan(&id, &visitorID, &productID, &createdAt); err != nil {
		return nil, err
	}
	return domain.RehydrateWishlistItem(id, visitorID, productID, createdAt), nil
}
