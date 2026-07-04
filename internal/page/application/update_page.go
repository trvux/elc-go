package application

import (
	"context"

	"github.com/trvux/elc-go/internal/page/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdatePage(ctx context.Context, repo domain.PageRepository, input domain.UpdatePageInput) (*domain.Page, error) {
	p, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperr.NewNotFoundError("page")
	}

	if err := p.Update(
		input.Title,
		input.Slug,
		input.Content,
		input.IsPublished,
		input.MetaTitle,
		input.MetaDescription,
		input.OrderIndex,
	); err != nil {
		return nil, err
	}

	return repo.Update(ctx, p)
}
