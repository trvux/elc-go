package application

import (
	"context"

	"github.com/trvux/elc-go/internal/recently-viewed/domain"
)

func ListRecentlyViewed(ctx context.Context, repo domain.RecentlyViewedRepository, visitorID string) ([]*domain.RecentlyViewedItemWithProduct, error) {
	return repo.List(ctx, visitorID)
}
