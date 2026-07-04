package domain

import "context"

type GroupRepository interface {
	GetAll(ctx context.Context, filter GroupFilter) ([]*Group, error)
	GetByID(ctx context.Context, id string) (*Group, error)
	GetBySlug(ctx context.Context, slug string) (*Group, error)
	Create(ctx context.Context, group *Group) (*Group, error)
	Update(ctx context.Context, group *Group) (*Group, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
