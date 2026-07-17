package domain

import "context"

type AttributeDefinitionRepository interface {
	GetAll(ctx context.Context, filter AttributeDefinitionFilter) ([]*AttributeDefinitionWithCategories, error)
	GetByID(ctx context.Context, id string) (*AttributeDefinitionWithCategories, error)
	// GetByIDs batch-loads plain definitions (no category association) for
	// a set of ids — used by product's application layer to validate
	// attribute value shapes and is_required on create/update, without an
	// N+1. Same cross-module read pattern as tag.GetByIDs.
	GetByIDs(ctx context.Context, ids []string) ([]*AttributeDefinition, error)
	Create(ctx context.Context, definition *AttributeDefinition) (*AttributeDefinition, error)
	Update(ctx context.Context, definition *AttributeDefinition) (*AttributeDefinition, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error

	// AttachCategories adds definitionID to each categoryIDs' attribute set
	// (additive — existing associations to other categories are untouched).
	// Duplicate attaches are a no-op.
	AttachCategories(ctx context.Context, definitionID string, categoryIDs []string) error
	// DetachCategory removes one association; the definition itself and its
	// associations with other categories are untouched.
	DetachCategory(ctx context.Context, definitionID, categoryID string) error
	// GetApplicableForCategory returns every non-deleted definition attached
	// to categoryID plus every global definition (zero category
	// associations) — same result AttributeDefinitionFilter{CategoryID,
	// IncludeGlobal: true} on GetAll produces, exposed directly for
	// call sites (like product's cross-module validation) that don't need
	// the full filter shape.
	GetApplicableForCategory(ctx context.Context, categoryID string) ([]*AttributeDefinition, error)
}
