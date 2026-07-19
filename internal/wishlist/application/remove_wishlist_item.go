package application

import (
	"context"

	"github.com/trvux/elc-go/internal/wishlist/domain"
)

func RemoveWishlistItem(ctx context.Context, repo domain.WishlistRepository, visitorID, productID string) error {
	return repo.Remove(ctx, visitorID, productID)
}
