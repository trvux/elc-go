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
	// FindOpenBySessionAndChannel returns the most recent still-open (new or
	// contacted) inquiry for this session+channel, or nil if none exists —
	// used to collapse repeated contact-channel clicks from the same
	// visitor into one lead instead of one row per click. See
	// application.RecordContactClick.
	FindOpenBySessionAndChannel(ctx context.Context, sessionID string, channel ContactChannel) (*Inquiry, error)
	// UpdateClickContext persists a fresh touch onto an existing click-
	// origin inquiry — see Inquiry.RefreshClickContext.
	UpdateClickContext(ctx context.Context, inquiry *Inquiry) (*Inquiry, error)
	// UpdateDetails persists name/phone/conversion_value changes made via
	// the entity's own SetIdentity/SetConversionValue mutators.
	UpdateDetails(ctx context.Context, inquiry *Inquiry) (*Inquiry, error)
}
