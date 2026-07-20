package application

import (
	"context"

	"github.com/trvux/elc-go/internal/review/domain"
)

// CreateReview persists a star rating/comment submitted from the public
// site.
func CreateReview(
	ctx context.Context,
	repo domain.ReviewRepository,
	input domain.CreateReviewInput,
) (*domain.Review, error) {
	review, err := domain.NewReview(
		input.Rating, input.Comment, input.ReviewerName, input.ReviewerPhone,
		input.ProductID, input.ProjectID, input.ServiceID, input.NewsID,
		input.SourceIP, input.UserAgent,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, review)
}
