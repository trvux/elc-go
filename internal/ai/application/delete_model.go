package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
)

func DeleteModel(ctx context.Context, repo domain.ModelRepository, id string) error {
	return repo.Delete(ctx, id)
}
