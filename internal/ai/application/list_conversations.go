package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
)

func ListConversations(ctx context.Context, repo domain.ConversationRepository, filter domain.ConversationFilter) ([]*domain.ConversationSummary, int, error) {
	return repo.ListConversations(ctx, filter)
}
