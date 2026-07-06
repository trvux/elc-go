package application

import (
	"context"

	"github.com/trvux/elc-go/internal/event/domain"
)

func GetTopViewed(ctx context.Context, repo domain.EventRepository, filter domain.TopViewedFilter) ([]domain.EntityViewCount, error) {
	return repo.TopViewed(ctx, filter)
}
