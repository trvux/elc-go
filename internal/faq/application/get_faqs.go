package application

import (
	"context"

	"github.com/trvux/elc-go/internal/faq/domain"
)

func GetFAQsByOwner(ctx context.Context, repo domain.FAQRepository, filter domain.FAQFilter) ([]*domain.FAQ, error) {
	return repo.GetByOwner(ctx, filter)
}

func GetFAQByID(ctx context.Context, repo domain.FAQRepository, id string) (*domain.FAQ, error) {
	return repo.GetByID(ctx, id)
}
