package domain

import "context"

type ServiceGroupRepository interface {
	GetAll(ctx context.Context, filter ServiceGroupFilter) ([]*ServiceGroup, error)
	GetByID(ctx context.Context, id string) (*ServiceGroup, error)
	GetBySlug(ctx context.Context, slug string) (*ServiceGroup, error)
	Create(ctx context.Context, serviceGroup *ServiceGroup) (*ServiceGroup, error)
	Update(ctx context.Context, serviceGroup *ServiceGroup) (*ServiceGroup, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
