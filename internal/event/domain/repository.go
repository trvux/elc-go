package domain

import "context"

type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	// TopViewed aggregates view_item counts per entity — used by the
	// dashboard's "most viewed" panel.
	TopViewed(ctx context.Context, filter TopViewedFilter) ([]EntityViewCount, error)
}
