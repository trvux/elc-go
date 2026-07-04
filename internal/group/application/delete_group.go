package application

import (
	"context"

	"github.com/trvux/elc-go/internal/group/domain"
)

func DeleteGroup(ctx context.Context, repo domain.GroupRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreGroup(ctx context.Context, repo domain.GroupRepository, id string) error {
	return repo.Restore(ctx, id)
}
