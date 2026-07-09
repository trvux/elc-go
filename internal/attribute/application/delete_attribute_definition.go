package application

import (
	"context"

	"github.com/trvux/elc-go/internal/attribute/domain"
)

func DeleteAttributeDefinition(ctx context.Context, repo domain.AttributeDefinitionRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreAttributeDefinition(ctx context.Context, repo domain.AttributeDefinitionRepository, id string) error {
	return repo.Restore(ctx, id)
}
