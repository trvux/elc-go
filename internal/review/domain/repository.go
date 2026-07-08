package domain

import "context"

// ReviewRepository persists customer reviews submitted from the public
// site and read by both the public product/service pages and the admin
// moderation screen.
type ReviewRepository interface {
	Create(ctx context.Context, review *Review) (*Review, error)
	GetAll(ctx context.Context, filter ReviewFilter) ([]*Review, error)
	Count(ctx context.Context, filter ReviewFilter) (int, error)
	GetByID(ctx context.Context, id string) (*Review, error)
	// Update persists IsPublished changes made via Review.SetPublished.
	Update(ctx context.Context, review *Review) (*Review, error)
	Delete(ctx context.Context, id string) error
	// GetSummary aggregates published reviews for exactly one of
	// productID/serviceID — used for star-rating display and
	// schema.org AggregateRating.
	GetSummary(ctx context.Context, productID, serviceID *string) (*ReviewSummary, error)
}
