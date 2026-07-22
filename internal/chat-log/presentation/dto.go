package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/chat-log/domain"
)

type createChatLogRequest struct {
	Message string `json:"message"`
	Kind    string `json:"kind"`
}

// chatLogResponse includes VisitorID — fine on this staff-only endpoint
// (see routes.go): it's the same anonymous, server-issued cookie value
// recently-viewed/wishlist already key off, not PII, and grouping messages
// by it is exactly what makes the logged data useful for later analysis
// (which messages came from the same shopper's session).
type chatLogResponse struct {
	ID        string    `json:"id"`
	VisitorID string    `json:"visitor_id"`
	Message   string    `json:"message"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

func toChatLogResponse(entry *domain.ChatLogEntry) chatLogResponse {
	return chatLogResponse{
		ID:        entry.ID(),
		VisitorID: entry.VisitorID(),
		Message:   entry.Message(),
		Kind:      entry.Kind(),
		CreatedAt: entry.CreatedAt(),
	}
}

func toChatLogResponseList(entries []*domain.ChatLogEntry) []chatLogResponse {
	out := make([]chatLogResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toChatLogResponse(e))
	}
	return out
}
