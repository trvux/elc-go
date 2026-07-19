package application

import (
	"context"

	"github.com/trvux/elc-go/internal/attribute/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// AttachAttributeDefinitionCategories attaches a definition to one or more
// categories — additive, existing associations with other categories are
// left untouched (see domain.AttributeDefinitionRepository.AttachCategories).
func AttachAttributeDefinitionCategories(ctx context.Context, repo domain.AttributeDefinitionRepository, definitionID string, categoryIDs []string) error {
	def, err := repo.GetByID(ctx, definitionID)
	if err != nil {
		return err
	}
	if def == nil {
		return apperr.NewNotFoundError("attribute_definition")
	}
	return repo.AttachCategories(ctx, definitionID, categoryIDs)
}

// DetachAttributeDefinitionCategory removes one category association; the
// definition and its other category associations are untouched.
func DetachAttributeDefinitionCategory(ctx context.Context, repo domain.AttributeDefinitionRepository, definitionID, categoryID string) error {
	return repo.DetachCategory(ctx, definitionID, categoryID)
}
