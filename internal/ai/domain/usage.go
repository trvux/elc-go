package domain

import "time"

// UsageGroupBy is how GetUsageReport buckets ai_messages — one of a fixed
// set of pre-written SQL queries (see the Postgres implementation), never a
// dynamically-built GROUP BY column, so an invalid value can't reach SQL at
// all.
type UsageGroupBy string

const (
	UsageGroupByDay      UsageGroupBy = "day"
	UsageGroupByProvider UsageGroupBy = "provider"
	UsageGroupByModel    UsageGroupBy = "model"
)

func (g UsageGroupBy) IsValid() bool {
	switch g {
	case UsageGroupByDay, UsageGroupByProvider, UsageGroupByModel:
		return true
	}
	return false
}

// UsageReportRow is one bucket of GetUsageReport's result — Key is a date
// (groupBy=day), a provider name (groupBy=provider), or
// "{providerName}/{modelName}" (groupBy=model).
type UsageReportRow struct {
	Key          string
	MessageCount int
	BlockedCount int
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// ConversationFilter scopes ListConversations — From/To nil means
// unbounded on that side.
type ConversationFilter struct {
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}

// ConversationSummary is ListConversations' row shape — a Conversation plus
// aggregates computed in SQL (message count, total cost), not fields on
// Conversation itself (those would be meaningless outside a list query).
type ConversationSummary struct {
	ID           string
	VisitorID    string
	UserID       *string
	MessageCount int
	TotalCostUSD float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
