package domain

import "context"

type FAQRepository interface {
	// GetByOwner returns FAQs for one owner, ordered by order_index ASC —
	// filter.PublishedOnly distinguishes the public read (published only)
	// from the admin view (everything, including drafts).
	GetByOwner(ctx context.Context, filter FAQFilter) ([]*FAQ, error)
	GetByID(ctx context.Context, id string) (*FAQ, error)
	Create(ctx context.Context, faq *FAQ) (*FAQ, error)
	Update(ctx context.Context, faq *FAQ) (*FAQ, error)
	// Delete is a hard delete — unlike catalog entities (products/services/
	// ...), a FAQ row has no independent SEO/URL value to preserve and no
	// downstream FK depends on it, so there's nothing worth a soft-delete
	// recovery window for.
	Delete(ctx context.Context, id string) error
}
