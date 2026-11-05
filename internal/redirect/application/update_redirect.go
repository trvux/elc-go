package application

import (
	"context"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/redirect/domain"
)

func UpdateRedirect(ctx context.Context, repo domain.Repository, input domain.UpdateRedirectInput) (*domain.Redirect, error) {
	r, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, apperr.NewNotFoundError("redirect")
	}

	retargeted := false
	if input.ToPath != nil {
		to, err := domain.NormalizeTarget(*input.ToPath)
		if err != nil {
			return nil, apperr.NewValidationError("validation failed", map[string][]string{"to_path": {"đích không hợp lệ"}})
		}
		final, err := resolveTarget(ctx, repo, r.FromPath(), to)
		if err != nil {
			return nil, err
		}
		if final != r.ToPath() {
			if err := r.Retarget(final); err != nil {
				return nil, err
			}
			retargeted = true
		}
	}
	if input.Note != nil {
		r.SetNote(*input.Note)
	}
	if input.IsActive != nil {
		r.SetActive(*input.IsActive)
	}

	updated, err := repo.Update(ctx, r)
	if err != nil {
		return nil, err
	}
	if retargeted {
		if err := repo.Repoint(ctx, updated.FromPath(), updated.ToPath()); err != nil {
			return nil, err
		}
	}
	return updated, nil
}
