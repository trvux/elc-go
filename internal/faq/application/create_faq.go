package application

import (
	"context"

	"github.com/trvux/elc-go/internal/faq/domain"
)

func CreateFAQ(ctx context.Context, repo domain.FAQRepository, input domain.CreateFAQInput) (*domain.FAQ, error) {
	faq, err := domain.NewFAQ(
		input.OwnerType, input.OwnerID, input.Question, input.Answer,
		input.OrderIndex, input.IsPublished,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, faq)
}
