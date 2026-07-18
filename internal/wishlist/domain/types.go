package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// ProductSummary is a lightweight, read-only reference to a product —
// resolved via a direct SQL join into `products`, same cross-module read
// pattern as product's own CategoryRef/BrandRef (see
// internal/product/domain/types.go). Not the full product.Product domain
// type, to avoid a cross-module Go dependency for the handful of display
// fields a wishlist list actually needs.
type ProductSummary struct {
	ID           string
	Name         string
	Slug         string
	ImageURL     string
	DisplayPrice *int64
}

// WishlistItem records that an anonymous visitor (identified by a
// server-issued visitor_id cookie, see internal/platform/httpserver's
// EnsureVisitorID — no customer accounts exist yet) saved a product. No
// customer/account concept is involved; visitor_id is a plain opaque string,
// not a foreign key to any user table.
type WishlistItem struct {
	id        string
	visitorID string
	productID string
	createdAt time.Time
}

// NewWishlistItem validates and creates a new entity from a request.
func NewWishlistItem(visitorID, productID string) (*WishlistItem, error) {
	fields := map[string][]string{}
	if visitorID == "" {
		fields["visitor_id"] = []string{"visitor_id is required"}
	}
	if productID == "" {
		fields["product_id"] = []string{"product_id is required"}
	}
	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	return &WishlistItem{
		visitorID: visitorID,
		productID: productID,
		createdAt: time.Now(),
	}, nil
}

// RehydrateWishlistItem reconstructs from a trusted DB row — no validation.
// Only the infrastructure layer should call this.
func RehydrateWishlistItem(id, visitorID, productID string, createdAt time.Time) *WishlistItem {
	return &WishlistItem{id: id, visitorID: visitorID, productID: productID, createdAt: createdAt}
}

func (w *WishlistItem) ID() string           { return w.id }
func (w *WishlistItem) VisitorID() string    { return w.visitorID }
func (w *WishlistItem) ProductID() string    { return w.productID }
func (w *WishlistItem) CreatedAt() time.Time { return w.createdAt }

// WishlistItemWithProduct is the read-shape List returns — a WishlistItem
// plus the joined product display summary.
type WishlistItemWithProduct struct {
	*WishlistItem
	Product *ProductSummary
}
