package application

import (
	"context"

	"github.com/trvux/elc-go/internal/attribute/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdateAttributeDefinition(ctx context.Context, repo domain.AttributeDefinitionRepository, input domain.UpdateAttributeDefinitionInput) (*domain.AttributeDefinition, error) {
	existing, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.NewNotFoundError("attribute_definition")
	}
	def := existing.AttributeDefinition

	name := def.Name()
	if input.Name != nil {
		name = *input.Name
	}
	groupLabel := def.GroupLabel()
	if input.GroupLabel != nil {
		groupLabel = input.GroupLabel
	}
	unit := def.Unit()
	if input.Unit != nil {
		unit = input.Unit
	}
	options := def.Options()
	if input.Options != nil {
		options = *input.Options
	}
	orderIndex := def.OrderIndex()
	if input.OrderIndex != nil {
		orderIndex = *input.OrderIndex
	}
	isRequired := def.IsRequired()
	if input.IsRequired != nil {
		isRequired = *input.IsRequired
	}

	if err := def.Update(name, groupLabel, unit, options, orderIndex, isRequired); err != nil {
		return nil, err
	}

	return repo.Update(ctx, def)
}
