package domain

import "context"

type BranchRepository interface {
	GetAll(ctx context.Context, filter BranchFilter) ([]*Branch, error)
	Count(ctx context.Context, filter BranchFilter) (int, error)
	GetByID(ctx context.Context, id string) (*Branch, error)
	GetBySlug(ctx context.Context, slug string) (*Branch, error)
	Create(ctx context.Context, branch *Branch) (*Branch, error)
	Update(ctx context.Context, branch *Branch) (*Branch, error)
	Delete(ctx context.Context, id string) error
}
