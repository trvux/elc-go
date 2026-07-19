package application

import (
	"context"

	"github.com/trvux/elc-go/internal/wishlist/domain"
)

func ListWishlistItems(ctx context.Context, repo domain.WishlistRepository, visitorID string) ([]*domain.WishlistItemWithProduct, error) {
	return repo.List(ctx, visitorID)
}
