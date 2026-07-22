package domain

import "context"

// ChatLogRepository persists shopper messages logged from the AI chat
// finder — write path (Create) is public/anonymous, read path (GetAll,
// Count) is staff-only (see presentation/routes.go).
type ChatLogRepository interface {
	Create(ctx context.Context, entry *ChatLogEntry) (*ChatLogEntry, error)
	GetAll(ctx context.Context, filter ChatLogFilter) ([]*ChatLogEntry, error)
	Count(ctx context.Context, filter ChatLogFilter) (int, error)
}
