package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
)

func DeleteProvider(ctx context.Context, repo domain.ProviderRepository, id string) error {
	return repo.Delete(ctx, id)
}
