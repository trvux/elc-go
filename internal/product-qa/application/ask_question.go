package application

import (
	"context"

	"github.com/trvux/elc-go/internal/product-qa/domain"
)

func AskQuestion(ctx context.Context, repo domain.QuestionRepository, input domain.CreateQuestionInput) (*domain.Question, error) {
	q, err := domain.NewQuestion(input.ProductID, input.AskerName, input.AskerEmail, input.QuestionText, input.SourceIP, input.UserAgent)
	if err != nil {
		return nil, err
	}
	return repo.Create(ctx, q)
}
