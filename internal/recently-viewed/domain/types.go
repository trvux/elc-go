package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// ProductSummary is a lightweight, read-only reference to a product —
// resolved via a direct SQL join into `products`, same cross-module read
// pattern as product's own CategoryRef/BrandRef (see
// internal/product/domain/types.go).
type ProductSummary struct {
	ID           string
	Name         string
	Slug         string
	ImageURL     string
	DisplayPrice *int64
}

// RecentlyViewedItem records that an anonymous visitor (identified by a
// server-issued visitor_id cookie, see internal/platform/httpserver's
// EnsureVisitorID — no customer accounts exist yet) viewed a product's page.
// Recording the same product again just bumps viewedAt (see
// RecentlyViewedRepository.Record) rather than creating a second row, so
// "recently viewed" always reflects each product's most recent view.
type RecentlyViewedItem struct {
	id        string
	visitorID string
	productID string
	viewedAt  time.Time
}

// NewRecentlyViewedItem validates and creates a new entity from a request.
func NewRecentlyViewedItem(visitorID, productID string) (*RecentlyViewedItem, error) {
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

	return &RecentlyViewedItem{
		visitorID: visitorID,
		productID: productID,
		viewedAt:  time.Now(),
	}, nil
}

// RehydrateRecentlyViewedItem reconstructs from a trusted DB row — no
// validation. Only the infrastructure layer should call this.
func RehydrateRecentlyViewedItem(id, visitorID, productID string, viewedAt time.Time) *RecentlyViewedItem {
	return &RecentlyViewedItem{id: id, visitorID: visitorID, productID: productID, viewedAt: viewedAt}
}

func (i *RecentlyViewedItem) ID() string          { return i.id }
func (i *RecentlyViewedItem) VisitorID() string   { return i.visitorID }
func (i *RecentlyViewedItem) ProductID() string   { return i.productID }
func (i *RecentlyViewedItem) ViewedAt() time.Time { return i.viewedAt }

// RecentlyViewedItemWithProduct is the read-shape List returns — an item
// plus the joined product display summary.
type RecentlyViewedItemWithProduct struct {
	*RecentlyViewedItem
	Product *ProductSummary
}
