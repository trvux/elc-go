package domain

import "context"

// WishlistRepository — Add is idempotent (adding a product already on the
// visitor's wishlist is a no-op that still returns the existing row, not an
// error) since the FE will call it as a plain toggle-on action.
type WishlistRepository interface {
	Add(ctx context.Context, item *WishlistItem) (*WishlistItem, error)
	Remove(ctx context.Context, visitorID, productID string) error
	List(ctx context.Context, visitorID string) ([]*WishlistItemWithProduct, error)
}
