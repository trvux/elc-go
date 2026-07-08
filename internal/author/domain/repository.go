package domain

import "context"

type AuthorRepository interface {
	GetAll(ctx context.Context, filter AuthorFilter) ([]*Author, error)
	GetByID(ctx context.Context, id string) (*Author, error)
	GetBySlug(ctx context.Context, slug string) (*Author, error)
	Create(ctx context.Context, author *Author) (*Author, error)
	Update(ctx context.Context, author *Author) (*Author, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
