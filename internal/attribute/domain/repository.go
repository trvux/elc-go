package domain

import "context"

type AttributeDefinitionRepository interface {
	GetAll(ctx context.Context, filter AttributeDefinitionFilter) ([]*AttributeDefinition, error)
	GetByID(ctx context.Context, id string) (*AttributeDefinition, error)
	// GetByIDs batch-loads definitions for a set of ids — used by product's
	// infrastructure layer to resolve product_attribute_values rows without
	// an N+1, same cross-module read pattern as tag.GetByIDs.
	GetByIDs(ctx context.Context, ids []string) ([]*AttributeDefinition, error)
	Create(ctx context.Context, definition *AttributeDefinition) (*AttributeDefinition, error)
	Update(ctx context.Context, definition *AttributeDefinition) (*AttributeDefinition, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
