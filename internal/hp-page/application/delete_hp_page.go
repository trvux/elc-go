package application

import (
	"context"

	"github.com/trvux/elc-go/internal/hp-page/domain"
)

func DeleteHpPage(ctx context.Context, repo domain.HpPageRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreHpPage(ctx context.Context, repo domain.HpPageRepository, id string) error {
	return repo.Restore(ctx, id)
}
