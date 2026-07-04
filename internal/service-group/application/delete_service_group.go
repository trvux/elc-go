package application

import (
	"context"

	"github.com/trvux/elc-go/internal/service-group/domain"
)

// DeleteServiceGroup soft-deletes — see docs/service-group.md for the
// services.group_id cleanup this cascades into at the repository level.
func DeleteServiceGroup(ctx context.Context, repo domain.ServiceGroupRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreServiceGroup(ctx context.Context, repo domain.ServiceGroupRepository, id string) error {
	return repo.Restore(ctx, id)
}
