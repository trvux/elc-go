package application

import (
	"context"
	"strings"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/redirect/domain"
)

// resolveTarget follows redirects starting at `to` until it reaches a path that
// does not itself redirect, so a new redirect never creates a chain. If the walk
// comes back to `from`, the new redirect would loop and is rejected.
func resolveTarget(ctx context.Context, repo domain.Repository, from, to string) (string, error) {
	current := to
	for i := 0; i < domain.MaxChainDepth; i++ {
		if strings.HasPrefix(current, "http") { // external destination, nothing to follow
			return current, nil
		}
		if current == from {
			return "", loopError()
		}
		next, err := repo.GetByFrom(ctx, current)
		if err != nil {
			return "", err
		}
		if next == nil || !next.IsActive() {
			return current, nil
		}
		current = next.ToPath()
	}
	return "", loopError()
}

func loopError() error {
	return apperr.NewValidationError("validation failed", map[string][]string{
		"to_path": {"đích này dẫn ngược về đường dẫn nguồn (vòng lặp chuyển hướng)"},
	})
}
