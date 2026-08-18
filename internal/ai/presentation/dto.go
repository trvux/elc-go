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
