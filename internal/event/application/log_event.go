package application

import (
	"context"

	"github.com/trvux/elc-go/internal/event/domain"
)

func LogEvent(ctx context.Context, repo domain.EventRepository, input domain.CreateEventInput) error {
	event, err := domain.NewEvent(input.Name, input.EntityType, input.EntityID, input.PagePath, input.SessionID)
	if err != nil {
		return err
	}
	return repo.Create(ctx, event)
}
