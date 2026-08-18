package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
)

func GetConversationMessages(ctx context.Context, repo domain.ConversationRepository, conversationID string) ([]*domain.ConversationMessage, error) {
	return repo.GetMessages(ctx, conversationID)
}
