package domain

import "context"

// InquiryRepository persists leads submitted from the public site.
type InquiryRepository interface {
	Create(ctx context.Context, inquiry *Inquiry) (*Inquiry, error)
	GetAll(ctx context.Context, filter InquiryFilter) ([]*Inquiry, error)
	Count(ctx context.Context, filter InquiryFilter) (int, error)
	GetByID(ctx context.Context, id string) (*Inquiry, error)
	// Update persists status/internal_note changes made via the entity's own
	// mutator methods (MarkContacted, Close, SetInternalNote, ...).
	Update(ctx context.Context, inquiry *Inquiry) (*Inquiry, error)
}
