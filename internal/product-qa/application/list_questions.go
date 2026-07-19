package application

import (
	"context"

	"github.com/trvux/elc-go/internal/product-qa/domain"
)

func ListQuestions(ctx context.Context, repo domain.QuestionRepository, filter domain.QuestionFilter) ([]*domain.Question, error) {
	return repo.GetAll(ctx, filter)
}

func CountQuestions(ctx context.Context, repo domain.QuestionRepository, filter domain.QuestionFilter) (int, error) {
	return repo.Count(ctx, filter)
}
