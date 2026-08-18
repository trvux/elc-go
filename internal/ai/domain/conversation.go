package domain

import "time"

// Conversation groups a visitor's messages. VisitorID (cookie-issued by
// httpserver.EnsureVisitorID, same as wishlist/recently-viewed) is the
// required key; UserID is attached opportunistically when the visitor
// happens to be logged in (via httpserver.OptionalAuth) — chat is never
// gated behind login, since that would shrink the conversation volume this
// exists to collect (see the Phase 1 RFC).
type Conversation struct {
	ID        string
	VisitorID string
	UserID    *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConversationMessage is one turn, persisted after the fact — append-only,
// no domain business rules to protect beyond what's already enforced by
// SendChatMessage's orchestration, so this is a plain value type (same
// spirit as ProductListResult) rather than a private-field entity.
type ConversationMessage struct {
	ID             string
	ConversationID string
	Role           Role
	Content        string
	// BlockedReason is set when the guardrail classifier rejected this
	// turn — Provider/Model/Usage/CostUSD are all nil in that case, since
	// no chat completion was ever called.
	BlockedReason *string
	ProviderID    *string
	ModelID       *string
	Usage         *TokenUsage
	CostUSD       *float64
	// ProductsShown is the slugs search_products surfaced this turn — feeds
	// the ads/marketing-planning analytics goal, not just support review.
	ProductsShown []string
	CreatedAt     time.Time
}
