package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/product-qa/domain"
)

func AnswerQuestion(ctx context.Context, repo domain.QuestionRepository, id, answerText string) (*domain.Question, error) {
	q, err := repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if q == nil {
		return nil, apperr.NewNotFoundError("question")
	}

	if err := q.Answer(answerText); err != nil {
		return nil, err
	}

	return repo.Update(ctx, q)
}
