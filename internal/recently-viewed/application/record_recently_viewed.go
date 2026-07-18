package application

import (
	"context"

	"github.com/trvux/elc-go/internal/recently-viewed/domain"
)

func RecordRecentlyViewed(ctx context.Context, repo domain.RecentlyViewedRepository, visitorID, productID string) (*domain.RecentlyViewedItem, error) {
	item, err := domain.NewRecentlyViewedItem(visitorID, productID)
	if err != nil {
		return nil, err
	}
	return repo.Record(ctx, item)
}
