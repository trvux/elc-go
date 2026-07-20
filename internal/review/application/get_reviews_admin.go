package application

import (
	"context"

	"github.com/trvux/elc-go/internal/review/domain"
)

// GetReviews and CountReviews back the staff-only admin list — every
// review regardless of entity type or is_published, unlike
// ListPublishedReviews which is scoped to one entity's published rows.
func GetReviews(ctx context.Context, repo domain.ReviewRepository, filter domain.ReviewFilter) ([]*domain.ReviewWithProduct, error) {
	return repo.GetAll(ctx, filter)
}

func CountReviews(ctx context.Context, repo domain.ReviewRepository, filter domain.ReviewFilter) (int, error) {
	return repo.Count(ctx, filter)
}
