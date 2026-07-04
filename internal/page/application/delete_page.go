package application

import (
	"context"

	"github.com/trvux/elc-go/internal/page/domain"
)

func DeletePage(ctx context.Context, repo domain.PageRepository, id string) error {
	return repo.Delete(ctx, id)
}

func RestorePage(ctx context.Context, repo domain.PageRepository, id string) error {
	return repo.Restore(ctx, id)
}
