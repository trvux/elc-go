package application

import (
	"context"

	"github.com/trvux/elc-go/internal/news/domain"
)

func CreateNews(ctx context.Context, repo domain.NewsRepository, input domain.CreateNewsInput) (*domain.News, error) {
	news, err := domain.NewNews(
		input.Title,
		input.Slug,
		input.Image,
		input.Content,
		input.CategoryID,
		input.IsPublished,
		input.MetaTitle,
		input.MetaDescription,
		input.OrderIndex,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, news)
}
