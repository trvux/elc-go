package presentation

import "github.com/trvux/elc-go/internal/ai/domain"

// maxHistoryMessages/maxMessageLength bound one chat request — a public,
// unauthenticated endpoint that fans out to a paid LLM API must not let a
// single caller pay for an unbounded conversation or an unbounded message.
const (
	maxHistoryMessages = 20
	maxMessageLength   = 2000
)

type chatMessageDTO struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Messages []chatMessageDTO `json:"messages"`
}

type chatResponse struct {
	Message chatMessageDTO `json:"message"`
}

func toDomainMessages(messages []chatMessageDTO) []domain.Message {
	out := make([]domain.Message, len(messages))
	for i, m := range messages {
		out[i] = domain.Message{Role: domain.Role(m.Role), Content: m.Content}
	}
	return out
}

func toChatResponse(m *domain.Message) chatResponse {
	return chatResponse{Message: chatMessageDTO{Role: string(m.Role), Content: m.Content}}
}
