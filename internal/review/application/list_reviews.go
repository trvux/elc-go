package application

import (
	"context"

	"github.com/trvux/elc-go/internal/review/domain"
)

// ListPublishedReviews returns one entity's published reviews (newest
// first) plus the aggregate rating computed over that same set.
func ListPublishedReviews(
	ctx context.Context,
	repo domain.ReviewRepository,
	entityColumn, entityID string,
) ([]*domain.Review, domain.ReviewAggregate, error) {
	reviews, err := repo.GetByEntity(ctx, entityColumn, entityID)
	if err != nil {
		return nil, domain.ReviewAggregate{}, err
	}

	if len(reviews) == 0 {
		return reviews, domain.ReviewAggregate{}, nil
	}

	sum := 0
	for _, r := range reviews {
		sum += r.Rating()
	}

	return reviews, domain.ReviewAggregate{
		Average: float64(sum) / float64(len(reviews)),
		Count:   len(reviews),
	}, nil
}
