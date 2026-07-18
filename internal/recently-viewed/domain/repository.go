package domain

import "context"

// RecentlyViewedRepository — Record is an upsert (viewing a product already
// in the list just bumps its viewedAt) and List always returns at most the
// last 20 by viewedAt, newest first — no separate cleanup job trims the
// table itself (see migrations/000001's doc comment).
type RecentlyViewedRepository interface {
	Record(ctx context.Context, item *RecentlyViewedItem) (*RecentlyViewedItem, error)
	List(ctx context.Context, visitorID string) ([]*RecentlyViewedItemWithProduct, error)
}
