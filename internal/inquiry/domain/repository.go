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

// LeadNotifier alerts internal staff that a new lead arrived. Mirrors
// auth.EmailSender's shape (one method per use case, not a generic Send) —
// see internal/auth/domain/repository.go. A notify failure must never fail
// lead capture: the lead is already durably saved via InquiryRepository, so
// losing the push is a missed notification, not a lost lead. Implementations
// (internal/inquiry/infrastructure) are expected to log their own failures
// and return nil.
type LeadNotifier interface {
	NotifyNewLead(ctx context.Context, inquiry *Inquiry) error
}

// ZaloFollowerRepository tracks which staff Zalo accounts have followed the
// company OA and messaged it — the recipients ZaloOASender pushes to.
type ZaloFollowerRepository interface {
	Upsert(ctx context.Context, follower *ZaloOAFollower) error
	SetActive(ctx context.Context, zaloUserID string, isActive bool) error
	GetAllActive(ctx context.Context) ([]*ZaloOAFollower, error)
}

// ZaloTokenRepository persists the single current OAuth token pair for the
// company's Zalo OA app (a singleton row — see the zalo_oa_tokens table).
type ZaloTokenRepository interface {
	// GetCurrent returns nil, nil if no token has ever been saved (Zalo OA
	// not bootstrapped yet via cmd/zalo-authorize).
	GetCurrent(ctx context.Context) (*ZaloOAToken, error)
	Save(ctx context.Context, token *ZaloOAToken) error
}
