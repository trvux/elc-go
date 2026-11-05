package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/redirect/domain"
)

func CreateRedirect(ctx context.Context, repo domain.Repository, input domain.CreateRedirectInput) (*domain.Redirect, error) {
	r, err := domain.NewRedirect(input.FromPath, input.ToPath, input.Note, domain.SourceManual)
	if err != nil {
		return nil, err
	}

	existing, err := repo.GetByFrom(ctx, r.FromPath())
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperr.NewConflictError("đã có chuyển hướng cho đường dẫn " + r.FromPath())
	}

	final, err := resolveTarget(ctx, repo, r.FromPath(), r.ToPath())
	if err != nil {
		return nil, err
	}
	if final != r.ToPath() {
		if err := r.Retarget(final); err != nil {
			return nil, err
		}
	}

	created, err := repo.Create(ctx, r)
	if err != nil {
		return nil, err
	}
	// Anything that used to land on this path now lands on the final target.
	if err := repo.Repoint(ctx, created.FromPath(), created.ToPath()); err != nil {
		return nil, err
	}
	return created, nil
}
