package application

import (
	"context"

	"github.com/trvux/elc-go/internal/product-qa/domain"
)

func DeleteQuestion(ctx context.Context, repo domain.QuestionRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}
