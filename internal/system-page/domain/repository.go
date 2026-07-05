package domain

import "context"

type SystemPageRepository interface {
	GetAll(ctx context.Context) ([]*SystemPage, error)
	GetByID(ctx context.Context, id string) (*SystemPage, error)
	GetBySlug(ctx context.Context, slug string) (*SystemPage, error)
	Update(ctx context.Context, page *SystemPage) (*SystemPage, error)
}
