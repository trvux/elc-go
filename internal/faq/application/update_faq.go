package application

import (
	"context"

	"github.com/trvux/elc-go/internal/faq/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdateFAQ(ctx context.Context, repo domain.FAQRepository, input domain.UpdateFAQInput) (*domain.FAQ, error) {
	faq, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if faq == nil {
		return nil, apperr.NewNotFoundError("faq")
	}

	if input.Question != nil {
		if err := faq.UpdateQuestion(*input.Question); err != nil {
			return nil, err
		}
	}
	if input.Answer != nil {
		if err := faq.UpdateAnswer(*input.Answer); err != nil {
			return nil, err
		}
	}
	if input.OrderIndex != nil {
		faq.Reorder(*input.OrderIndex)
	}
	if input.IsPublished != nil {
		faq.SetPublished(*input.IsPublished)
	}

	return repo.Update(ctx, faq)
}
