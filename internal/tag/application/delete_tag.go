package application

import (
	"context"

	"github.com/trvux/elc-go/internal/tag/domain"
)

func DeleteTag(ctx context.Context, repo domain.TagRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreTag(ctx context.Context, repo domain.TagRepository, id string) error {
	return repo.Restore(ctx, id)
}
