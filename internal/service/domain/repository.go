package domain

import "context"

type ServiceRepository interface {
	Count(ctx context.Context, filter ServiceFilter) (int, error)
	GetAll(ctx context.Context, filter ServiceFilter) ([]*ServiceWithRelations, error)
	GetByID(ctx context.Context, id string) (*ServiceWithRelations, error)
	GetBySlug(ctx context.Context, slug string) (*ServiceWithRelations, error)
	Create(ctx context.Context, service *Service) (*Service, error)
	Update(ctx context.Context, service *Service) (*Service, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	// SetCategories replaces the product categories a service is linked to.
	SetCategories(ctx context.Context, serviceID string, categoryIDs []string) error
}
