package domain

import (
	"context"
)

type PageFilter struct {
	IsPublished    *bool
	Search         string
	Limit          int
	Offset         int
	IncludeDeleted bool
}

type PageRepository interface {
	GetAll(ctx context.Context, filter PageFilter) ([]*Page, error)
	Count(ctx context.Context, filter PageFilter) (int, error)
	GetByID(ctx context.Context, id string) (*Page, error)
	GetBySlug(ctx context.Context, slug string) (*Page, error)
	Create(ctx context.Context, page *Page) (*Page, error)
	Update(ctx context.Context, page *Page) (*Page, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
