package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/ai/domain"
)

// maxMessageLength bounds one chat turn — a public, unauthenticated
// endpoint that fans out to a paid LLM API must not let a single caller pay
// for an unbounded message.
const maxMessageLength = 2000

// chatHistoryLimit is how many prior turns ConversationRepository.
// ListRecentMessages loads to give the model context — the client only
// ever sends the newest message (see chatRequest); the server owns history
// now that it's persisted.
const chatHistoryLimit = 20

type chatRequest struct {
	Message string `json:"message"`
}

type chatMessageDTO struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message chatMessageDTO `json:"message"`
	// Blocked is true when the guardrail classifier rejected this message
	// — Message.Content is then the canned refusal, not a model reply.
	Blocked bool `json:"blocked,omitempty"`
}

func toChatMessageDTO(role domain.Role, content string) chatMessageDTO {
	return chatMessageDTO{Role: string(role), Content: content}
}

// --- admin: providers ---

type providerResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	BaseURL       string `json:"baseUrl"`
	PricingDocURL string `json:"pricingDocUrl,omitempty"`
	// HasAPIKey tells the admin panel a key is configured without ever
	// round-tripping the plaintext — see domain.Provider's doc comment.
	HasAPIKey bool      `json:"hasApiKey"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func toProviderResponse(p *domain.Provider) providerResponse {
	return providerResponse{
		ID: p.ID(), Name: p.Name(), DisplayName: p.DisplayName(), BaseURL: p.BaseURL(),
		PricingDocURL: p.PricingDocURL(), HasAPIKey: true, IsActive: p.IsActive(),
		CreatedAt: p.CreatedAt(), UpdatedAt: p.UpdatedAt(),
	}
}

func toProviderResponseList(providers []*domain.Provider) []providerResponse {
	out := make([]providerResponse, len(providers))
	for i, p := range providers {
		out[i] = toProviderResponse(p)
	}
	return out
}

type createProviderRequest struct {
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	BaseURL       string `json:"baseUrl"`
	APIKey        string `json:"apiKey"`
	PricingDocURL string `json:"pricingDocUrl"`
}

type updateProviderRequest struct {
	DisplayName   *string `json:"displayName"`
	BaseURL       *string `json:"baseUrl"`
	APIKey        *string `json:"apiKey"`
	PricingDocURL *string `json:"pricingDocUrl"`
	IsActive      *bool   `json:"isActive"`
}

// --- admin: models ---

type modelResponse struct {
	ID               string         `json:"id"`
	ProviderID       string         `json:"providerId"`
	ModelName        string         `json:"modelName"`
	DisplayName      string         `json:"displayName"`
	Role             string         `json:"role"`
	Pricing          domain.Pricing `json:"pricing"`
	FallbackPriority int            `json:"fallbackPriority"`
	IsDefault        bool           `json:"isDefault"`
	IsActive         bool           `json:"isActive"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

func toModelResponse(m *domain.Model) modelResponse {
	return modelResponse{
		ID: m.ID(), ProviderID: m.ProviderID(), ModelName: m.ModelName(), DisplayName: m.DisplayName(),
		Role: string(m.Role()), Pricing: m.Pricing(), FallbackPriority: m.FallbackPriority(),
		IsDefault: m.IsDefault(), IsActive: m.IsActive(), CreatedAt: m.CreatedAt(), UpdatedAt: m.UpdatedAt(),
	}
}

func toModelResponseList(models []*domain.Model) []modelResponse {
	out := make([]modelResponse, len(models))
	for i, m := range models {
		out[i] = toModelResponse(m)
	}
	return out
}

type createModelRequest struct {
	ProviderID       string         `json:"providerId"`
	ModelName        string         `json:"modelName"`
	DisplayName      string         `json:"displayName"`
	Role             string         `json:"role"`
	Pricing          domain.Pricing `json:"pricing"`
	FallbackPriority int            `json:"fallbackPriority"`
}

type updateModelRequest struct {
	DisplayName      *string         `json:"displayName"`
	Pricing          *domain.Pricing `json:"pricing"`
	FallbackPriority *int            `json:"fallbackPriority"`
	IsDefault        *bool           `json:"isDefault"`
	IsActive         *bool           `json:"isActive"`
}

// --- admin: reporting (Phase 3) ---

const (
	defaultConversationListLimit = 20
	maxConversationListLimit     = 100
)

type usageReportRowDTO struct {
	Key          string  `json:"key"`
	MessageCount int     `json:"messageCount"`
	BlockedCount int     `json:"blockedCount"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUsd"`
}

type usageReportResponse struct {
	GroupBy string              `json:"groupBy"`
	From    time.Time           `json:"from"`
	To      time.Time           `json:"to"`
	Rows    []usageReportRowDTO `json:"rows"`
}

func toUsageReportResponse(groupBy string, from, to time.Time, rows []domain.UsageReportRow) usageReportResponse {
	out := make([]usageReportRowDTO, len(rows))
	for i, r := range rows {
		out[i] = usageReportRowDTO{
			Key: r.Key, MessageCount: r.MessageCount, BlockedCount: r.BlockedCount,
			InputTokens: r.InputTokens, OutputTokens: r.OutputTokens, CostUSD: r.CostUSD,
		}
	}
	return usageReportResponse{GroupBy: groupBy, From: from, To: to, Rows: out}
}

type conversationSummaryDTO struct {
	ID           string    `json:"id"`
	VisitorID    string    `json:"visitorId"`
	UserID       *string   `json:"userId,omitempty"`
	MessageCount int       `json:"messageCount"`
	TotalCostUSD float64   `json:"totalCostUsd"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type conversationListResponse struct {
	Conversations []conversationSummaryDTO `json:"conversations"`
	Total         int                      `json:"total"`
	Limit         int                      `json:"limit"`
	Offset        int                      `json:"offset"`
}

func toConversationListResponse(summaries []*domain.ConversationSummary, total, limit, offset int) conversationListResponse {
	out := make([]conversationSummaryDTO, len(summaries))
	for i, s := range summaries {
		out[i] = conversationSummaryDTO{
			ID: s.ID, VisitorID: s.VisitorID, UserID: s.UserID, MessageCount: s.MessageCount,
			TotalCostUSD: s.TotalCostUSD, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
		}
	}
	return conversationListResponse{Conversations: out, Total: total, Limit: limit, Offset: offset}
}

type tokenUsageDTO struct {
	InputTokens    int `json:"inputTokens"`
	OutputTokens   int `json:"outputTokens"`
	CacheHitTokens int `json:"cacheHitTokens"`
}

type conversationMessageDetailDTO struct {
	ID            string         `json:"id"`
	Role          string         `json:"role"`
	Content       string         `json:"content"`
	BlockedReason *string        `json:"blockedReason,omitempty"`
	Incomplete    bool           `json:"incomplete,omitempty"`
	ProviderID    *string        `json:"providerId,omitempty"`
	ModelID       *string        `json:"modelId,omitempty"`
	Usage         *tokenUsageDTO `json:"usage,omitempty"`
	CostUSD       *float64       `json:"costUsd,omitempty"`
	ProductsShown []string       `json:"productsShown,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
}

func toConversationMessageDetailList(messages []*domain.ConversationMessage) []conversationMessageDetailDTO {
	out := make([]conversationMessageDetailDTO, len(messages))
	for i, m := range messages {
		dto := conversationMessageDetailDTO{
			ID: m.ID, Role: string(m.Role), Content: m.Content, BlockedReason: m.BlockedReason,
			Incomplete: m.Incomplete, ProviderID: m.ProviderID, ModelID: m.ModelID,
			CostUSD: m.CostUSD, ProductsShown: m.ProductsShown, CreatedAt: m.CreatedAt,
		}
		if m.Usage != nil {
			dto.Usage = &tokenUsageDTO{InputTokens: m.Usage.InputTokens, OutputTokens: m.Usage.OutputTokens, CacheHitTokens: m.Usage.CacheHitTokens}
		}
		out[i] = dto
	}
	return out
}
