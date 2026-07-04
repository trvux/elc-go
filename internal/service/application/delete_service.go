package application

import (
	"context"

	"github.com/trvux/elc-go/internal/service/domain"
)

func DeleteService(ctx context.Context, repo domain.ServiceRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreService(ctx context.Context, repo domain.ServiceRepository, id string) error {
	return repo.Restore(ctx, id)
}
