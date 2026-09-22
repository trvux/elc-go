package application

import (
	"context"

	"github.com/trvux/elc-go/internal/faq/domain"
)

func DeleteFAQ(ctx context.Context, repo domain.FAQRepository, id string) error {
	return repo.Delete(ctx, id)
}
