package application

import (
	"context"

	"github.com/trvux/elc-go/internal/news/domain"
)

func GetNews(ctx context.Context, repo domain.NewsRepository, filter domain.NewsFilter) ([]*domain.News, error) {
	return repo.GetAll(ctx, filter)
}

func CountNews(ctx context.Context, repo domain.NewsRepository, filter domain.NewsFilter) (int, error) {
	return repo.Count(ctx, filter)
}

func GetNewsByID(ctx context.Context, repo domain.NewsRepository, id string) (*domain.News, error) {
	return repo.GetByID(ctx, id)
}

func GetNewsBySlug(ctx context.Context, repo domain.NewsRepository, slug string) (*domain.News, error) {
	return repo.GetBySlug(ctx, slug)
}
