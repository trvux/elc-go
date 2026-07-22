package application

import (
	"context"

	"github.com/trvux/elc-go/internal/chat-log/domain"
)

func CreateChatLog(ctx context.Context, repo domain.ChatLogRepository, visitorID, message, kind string) (*domain.ChatLogEntry, error) {
	entry, err := domain.NewChatLogEntry(visitorID, message, kind)
	if err != nil {
		return nil, err
	}
	return repo.Create(ctx, entry)
}
