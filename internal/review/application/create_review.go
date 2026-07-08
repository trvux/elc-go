package application

import (
	"context"

	"github.com/trvux/elc-go/internal/review/domain"
)

func CreateReview(ctx context.Context, repo domain.ReviewRepository, input domain.CreateReviewInput) (*domain.Review, error) {
	review, err := domain.NewReview(
		input.ProductID, input.ServiceID,
		input.Rating, input.Comment, input.ReviewerName, input.ReviewerPhone,
		input.SourceIP, input.UserAgent,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, review)
}
