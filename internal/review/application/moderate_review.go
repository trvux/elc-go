package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/review/domain"
)

// SetReviewPublished implements the admin moderation screen's show/hide
// action — a review that tripped the blocklist (false positive) can be
// restored, or one that slipped through published can be hidden.
func SetReviewPublished(ctx context.Context, repo domain.ReviewRepository, id string, isPublished bool) (*domain.Review, error) {
	review, err := repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if review == nil {
		return nil, apperr.NewNotFoundError("review")
	}

	review.SetPublished(isPublished)
	return repo.Update(ctx, review)
}

func DeleteReview(ctx context.Context, repo domain.ReviewRepository, id string) error {
	return repo.Delete(ctx, id)
}
