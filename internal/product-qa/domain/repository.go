package domain

import "context"

type QuestionRepository interface {
	Create(ctx context.Context, question *Question) (*Question, error)
	GetByID(ctx context.Context, id string) (*Question, error)
	GetAll(ctx context.Context, filter QuestionFilter) ([]*Question, error)
	Count(ctx context.Context, filter QuestionFilter) (int, error)
	Update(ctx context.Context, question *Question) (*Question, error)
	SoftDelete(ctx context.Context, id string) error
}
