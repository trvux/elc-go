package application

import (
	"context"

	"github.com/trvux/elc-go/internal/page/domain"
)

func CreatePage(ctx context.Context, repo domain.PageRepository, input domain.CreatePageInput) (*domain.Page, error) {
	p, err := domain.NewPage(
		input.Title,
		input.Slug,
		input.Content,
		input.IsPublished,
		input.MetaTitle,
		input.MetaDescription,
		input.OrderIndex,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, p)
}
