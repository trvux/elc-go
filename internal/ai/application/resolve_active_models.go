package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
)

// ResolveActiveModels returns the active models of role, in the exact
// fallback order SendChatMessage tries them — a thin wrapper so callers
// depend on this package rather than reaching into the repository
// directly.
func ResolveActiveModels(ctx context.Context, repo domain.ModelRepository, role domain.ModelRole) ([]domain.ModelConfig, error) {
	return repo.ListActiveConfigsByRole(ctx, role)
}
