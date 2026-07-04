package domain

import "context"

type BrandRepository interface {
	GetAll(ctx context.Context, filter BrandFilter) ([]*Brand, error)
	GetByID(ctx context.Context, id string) (*Brand, error)
	GetBySlug(ctx context.Context, slug string) (*Brand, error)
	Create(ctx context.Context, brand *Brand) (*Brand, error)
	Update(ctx context.Context, brand *Brand) (*Brand, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
