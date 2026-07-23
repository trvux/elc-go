package application

import (
	"context"

	"github.com/trvux/elc-go/internal/hp-page/domain"
)

func CreateHpPage(ctx context.Context, repo domain.HpPageRepository, input domain.CreateHpPageInput) (*domain.HpPage, error) {
	page, err := domain.NewHpPage(
		input.Name,
		input.Slug,
		input.ImageURL,
		input.MetaTitle,
		input.MetaDescription,
		input.OrderIndex,
		input.Content,
		input.AttributeCode,
		input.AttributeValues,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, page)
}
