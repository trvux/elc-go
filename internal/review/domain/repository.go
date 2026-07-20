package domain

import "context"

// ReviewRepository persists public star ratings/comments submitted from the
// site.
type ReviewRepository interface {
	Create(ctx context.Context, review *Review) (*Review, error)
	// GetByEntity returns published reviews for one entity, newest first.
	// entityColumn is one of "product_id"/"project_id"/"service_id"/
	// "news_id" — chosen by the caller from a fixed whitelist (see
	// presentation.entityColumnFor), never raw request input.
	GetByEntity(ctx context.Context, entityColumn, entityID string) ([]*Review, error)
	// GetAll/Count back the staff-only admin list — every review regardless
	// of entity type or is_published, newest first, with its product
	// name/slug joined in when it's a product review.
	GetAll(ctx context.Context, filter ReviewFilter) ([]*ReviewWithProduct, error)
	Count(ctx context.Context, filter ReviewFilter) (int, error)
}
