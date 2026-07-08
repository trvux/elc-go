package application

import (
	"context"

	"github.com/trvux/elc-go/internal/review/domain"
)

func GetReviews(ctx context.Context, repo domain.ReviewRepository, filter domain.ReviewFilter) ([]*domain.Review, error) {
	return repo.GetAll(ctx, filter)
}

func CountReviews(ctx context.Context, repo domain.ReviewRepository, filter domain.ReviewFilter) (int, error) {
	return repo.Count(ctx, filter)
}

func GetReviewByID(ctx context.Context, repo domain.ReviewRepository, id string) (*domain.Review, error) {
	return repo.GetByID(ctx, id)
}

func GetReviewSummary(ctx context.Context, repo domain.ReviewRepository, productID, serviceID *string) (*domain.ReviewSummary, error) {
	return repo.GetSummary(ctx, productID, serviceID)
}
