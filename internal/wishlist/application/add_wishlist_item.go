package application

import (
	"context"

	"github.com/trvux/elc-go/internal/wishlist/domain"
)

func AddWishlistItem(ctx context.Context, repo domain.WishlistRepository, visitorID, productID string) (*domain.WishlistItem, error) {
	item, err := domain.NewWishlistItem(visitorID, productID)
	if err != nil {
		return nil, err
	}
	return repo.Add(ctx, item)
}
