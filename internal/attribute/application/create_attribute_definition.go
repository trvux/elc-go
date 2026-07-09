package application

import (
	"context"

	"github.com/trvux/elc-go/internal/attribute/domain"
)

func CreateAttributeDefinition(ctx context.Context, repo domain.AttributeDefinitionRepository, input domain.CreateAttributeDefinitionInput) (*domain.AttributeDefinition, error) {
	def, err := domain.NewAttributeDefinition(
		input.CategoryID, input.Code, input.Name, input.GroupLabel,
		input.DataType, input.Unit, input.Options, input.OrderIndex, input.IsRequired,
	)
	if err != nil {
		return nil, err
	}
	return repo.Create(ctx, def)
}
