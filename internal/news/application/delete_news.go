package application

import (
	"context"

	"github.com/trvux/elc-go/internal/news/domain"
)

func DeleteNews(ctx context.Context, repo domain.NewsRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreNews(ctx context.Context, repo domain.NewsRepository, id string) error {
	return repo.Restore(ctx, id)
}
