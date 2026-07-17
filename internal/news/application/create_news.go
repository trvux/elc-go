package application

import (
	"context"

	"github.com/trvux/elc-go/internal/news/domain"
)

func CreateNews(ctx context.Context, repo domain.NewsRepository, input domain.CreateNewsInput) (*domain.News, error) {
	news, err := domain.NewNews(
		input.Title,
		input.Slug,
		input.Images,
		input.Content,
		input.Excerpt,
		input.CategoryID,
		input.AuthorID,
		input.IsPublished,
		input.MetaTitle,
		input.MetaDescription,
		input.OrderIndex,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, news, input.TagIDs)
}
