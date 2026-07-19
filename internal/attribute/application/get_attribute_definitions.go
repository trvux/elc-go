package application

import (
	"context"

	"github.com/trvux/elc-go/internal/attribute/domain"
)

func GetAttributeDefinitions(ctx context.Context, repo domain.AttributeDefinitionRepository, filter domain.AttributeDefinitionFilter) ([]*domain.AttributeDefinitionWithCategories, error) {
	return repo.GetAll(ctx, filter)
}

func GetAttributeDefinitionByID(ctx context.Context, repo domain.AttributeDefinitionRepository, id string) (*domain.AttributeDefinitionWithCategories, error) {
	return repo.GetByID(ctx, id)
}
