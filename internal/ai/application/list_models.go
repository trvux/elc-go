package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
)

func ListModels(ctx context.Context, repo domain.ModelRepository) ([]*domain.Model, error) {
	return repo.List(ctx)
}
