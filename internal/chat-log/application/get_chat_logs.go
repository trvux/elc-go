package application

import (
	"context"

	"github.com/trvux/elc-go/internal/chat-log/domain"
)

func GetChatLogs(ctx context.Context, repo domain.ChatLogRepository, filter domain.ChatLogFilter) ([]*domain.ChatLogEntry, error) {
	return repo.GetAll(ctx, filter)
}

func CountChatLogs(ctx context.Context, repo domain.ChatLogRepository, filter domain.ChatLogFilter) (int, error) {
	return repo.Count(ctx, filter)
}
