package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
)

func ListProviders(ctx context.Context, repo domain.ProviderRepository) ([]*domain.Provider, error) {
	return repo.List(ctx)
}
