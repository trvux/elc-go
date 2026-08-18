package domain

import (
	"context"
	"time"
)

// ProviderRepository is admin CRUD for configured LLM vendors. Create/
// Update take/return plaintext-in-memory Provider (see Provider's doc
// comment) — the Postgres implementation is responsible for encrypting
// APIKey before it ever reaches SQL and never decrypting it back out for
// List/GetByID (those come back with APIKey == "", so presentation can
// never accidentally round-trip a secret).
type ProviderRepository interface {
	Create(ctx context.Context, p *Provider) (*Provider, error)
	// Update persists p's non-secret fields. newAPIKey, if non-nil, replaces
	// the encrypted key; nil leaves the stored key untouched. This is a
	// separate parameter rather than reading p.APIKey() because GetByID/List
	// never populate APIKey() (see Provider's doc comment) — there is no way
	// to tell "caller means unchanged" from "caller means clear it" by
	// looking at the entity alone once it's round-tripped through a read.
	Update(ctx context.Context, p *Provider, newAPIKey *string) (*Provider, error)
	List(ctx context.Context) ([]*Provider, error)
	GetByID(ctx context.Context, id string) (*Provider, error)
	Delete(ctx context.Context, id string) error
}

// ModelRepository is admin CRUD for provider models plus the two
// runtime-facing reads SendChatMessage and cmd/sync-ai-pricing actually
// need.
type ModelRepository interface {
	Create(ctx context.Context, m *Model) (*Model, error)
	Update(ctx context.Context, m *Model) (*Model, error)
	List(ctx context.Context) ([]*Model, error)
	GetByID(ctx context.Context, id string) (*Model, error)
	Delete(ctx context.Context, id string) error
	// ListActiveConfigsByRole resolves every active model of role, joined
	// with its (active) provider's base URL and decrypted API key, ordered
	// by fallback_priority ascending — exactly the set/order
	// SendChatMessage's fallback loop tries in turn.
	ListActiveConfigsByRole(ctx context.Context, role ModelRole) ([]ModelConfig, error)
	// UpdatePricing is used only by cmd/sync-ai-pricing: updates the
	// pricing column for an existing (providerID, modelName) row. Returns
	// found=false and does nothing if no such row exists — sync never
	// creates a model, see the Phase 1 RFC.
	UpdatePricing(ctx context.Context, providerID, modelName string, pricing Pricing) (found bool, err error)
}

// ConversationRepository persists chat turns.
type ConversationRepository interface {
	// GetOrCreateByVisitor returns visitorID's most recent conversation if
	// one was updated recently enough to still count as the same session
	// (see the Postgres implementation's doc comment for the exact
	// threshold), or starts a new one. userID is attached when non-nil and
	// the conversation doesn't already have one.
	GetOrCreateByVisitor(ctx context.Context, visitorID string, userID *string) (*Conversation, error)
	// AppendMessage inserts msg and bumps its conversation's UpdatedAt.
	AppendMessage(ctx context.Context, msg *ConversationMessage) (*ConversationMessage, error)
	// ListRecentMessages returns up to limit of the conversation's most
	// recent user/assistant turns (oldest first), excluding blocked ones —
	// what SendChatMessage sends the model as prior context. Never includes
	// role="tool" rows: tool calls aren't persisted as their own messages
	// (see AppendMessage's callers), only the user/assistant turns are.
	ListRecentMessages(ctx context.Context, conversationID string, limit int) ([]*ConversationMessage, error)

	// ListConversations is the admin-facing paginated list (Phase 3) —
	// unlike ListRecentMessages, this is for a human reviewing history, not
	// for feeding a model, so it applies no role/blocked/incomplete filter.
	// Returns the total match count (ignoring Limit/Offset) for pagination.
	ListConversations(ctx context.Context, filter ConversationFilter) ([]*ConversationSummary, int, error)
	// GetMessages returns every message of one conversation, oldest first —
	// including blocked and incomplete ones, so an admin sees exactly what
	// happened, not the trimmed view ListRecentMessages feeds back to the
	// model.
	GetMessages(ctx context.Context, conversationID string) ([]*ConversationMessage, error)
	// GetUsageReport aggregates ai_messages (role=assistant only — user
	// turns carry no cost/tokens) between from and to, bucketed by groupBy.
	GetUsageReport(ctx context.Context, from, to time.Time, groupBy UsageGroupBy) ([]UsageReportRow, error)
}
