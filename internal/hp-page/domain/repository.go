package domain

import "context"

type HpPageRepository interface {
	GetAll(ctx context.Context, filter HpPageFilter) ([]*HpPage, error)
	GetByID(ctx context.Context, id string) (*HpPage, error)
	GetBySlug(ctx context.Context, slug string) (*HpPage, error)
	Create(ctx context.Context, page *HpPage) (*HpPage, error)
	Update(ctx context.Context, page *HpPage) (*HpPage, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
