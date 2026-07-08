package application

import (
	"context"

	"github.com/trvux/elc-go/internal/news/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdateNews(ctx context.Context, repo domain.NewsRepository, input domain.UpdateNewsInput) (*domain.News, error) {
	news, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if news == nil {
		return nil, apperr.NewNotFoundError("news")
	}

	if input.Title != nil {
		if err := news.UpdateTitle(*input.Title); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := news.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.Images != nil {
		news.UpdateImages(input.Images)
	}
	if input.Content != nil {
		news.UpdateContent(input.Content)
	}
	if input.Excerpt != nil {
		news.UpdateExcerpt(*input.Excerpt)
	}
	if input.CategoryID != nil {
		news.UpdateCategoryID(input.CategoryID)
	}
	if input.AuthorID != nil {
		news.UpdateAuthorID(input.AuthorID)
	}
	if input.IsPublished != nil {
		news.SetPublished(*input.IsPublished)
	}
	if input.MetaTitle != nil {
		if err := news.UpdateMetaTitle(input.MetaTitle); err != nil {
			return nil, err
		}
	}
	if input.MetaDescription != nil {
		if err := news.UpdateMetaDescription(input.MetaDescription); err != nil {
			return nil, err
		}
	}
	if input.Seo != nil {
		news.UpdateSeo(*input.Seo)
	}
	if input.OrderIndex != nil {
		news.Reorder(*input.OrderIndex)
	}

	return repo.Update(ctx, news, input.TagIDs)
}
