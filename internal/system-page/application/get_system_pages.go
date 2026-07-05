package application

import (
	"context"

	"github.com/trvux/elc-go/internal/system-page/domain"
)

func GetSystemPages(ctx context.Context, repo domain.SystemPageRepository) ([]*domain.SystemPage, error) {
	return repo.GetAll(ctx)
}

func GetSystemPageBySlug(ctx context.Context, repo domain.SystemPageRepository, slug string) (*domain.SystemPage, error) {
	return repo.GetBySlug(ctx, slug)
}
