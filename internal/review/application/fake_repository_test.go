package application

import (
	"context"
	"fmt"

	"github.com/trvux/elc-go/internal/review/domain"
)

// fakeReviewRepository is an in-memory stand-in for
// PostgresReviewRepository, used only in tests so the application layer can
// be tested without a real DB.
type fakeReviewRepository struct {
	reviews map[string]*domain.Review
}

func newFakeReviewRepository() *fakeReviewRepository {
	return &fakeReviewRepository{reviews: map[string]*domain.Review{}}
}

func (r *fakeReviewRepository) Create(ctx context.Context, review *domain.Review) (*domain.Review, error) {
	id := fmt.Sprintf("id-%d", len(r.reviews)+1)
	created := domain.RehydrateReview(
		id, review.ProductID(), review.ServiceID(),
		review.Rating(), review.Comment(), review.ReviewerName(), review.ReviewerPhone(),
		review.IsPublished(), review.SourceIP(), review.UserAgent(),
		review.CreatedAt(), review.UpdatedAt(),
	)
	r.reviews[id] = created
	return created, nil
}

func (r *fakeReviewRepository) GetAll(ctx context.Context, filter domain.ReviewFilter) ([]*domain.Review, error) {
	result := make([]*domain.Review, 0, len(r.reviews))
	for _, rv := range r.reviews {
		if filter.ProductID != nil && (rv.ProductID() == nil || *rv.ProductID() != *filter.ProductID) {
			continue
		}
		if filter.ServiceID != nil && (rv.ServiceID() == nil || *rv.ServiceID() != *filter.ServiceID) {
			continue
		}
		if filter.IsPublished != nil && rv.IsPublished() != *filter.IsPublished {
			continue
		}
		result = append(result, rv)
	}
	return result, nil
}

func (r *fakeReviewRepository) Count(ctx context.Context, filter domain.ReviewFilter) (int, error) {
	all, _ := r.GetAll(ctx, filter)
	return len(all), nil
}

func (r *fakeReviewRepository) GetByID(ctx context.Context, id string) (*domain.Review, error) {
	rv, ok := r.reviews[id]
	if !ok {
		return nil, nil
	}
	return rv, nil
}

func (r *fakeReviewRepository) Update(ctx context.Context, review *domain.Review) (*domain.Review, error) {
	r.reviews[review.ID()] = review
	return review, nil
}

func (r *fakeReviewRepository) Delete(ctx context.Context, id string) error {
	delete(r.reviews, id)
	return nil
}

func (r *fakeReviewRepository) GetSummary(ctx context.Context, productID, serviceID *string) (*domain.ReviewSummary, error) {
	isPublished := true
	all, _ := r.GetAll(ctx, domain.ReviewFilter{ProductID: productID, ServiceID: serviceID, IsPublished: &isPublished})
	if len(all) == 0 {
		return &domain.ReviewSummary{}, nil
	}
	sum := 0
	for _, rv := range all {
		sum += rv.Rating()
	}
	return &domain.ReviewSummary{Count: len(all), Average: float64(sum) / float64(len(all))}, nil
}
