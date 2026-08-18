package application

import (
	"context"

	"github.com/trvux/elc-go/internal/branch/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdateBranch(ctx context.Context, repo domain.BranchRepository, input domain.UpdateBranchInput) (*domain.Branch, error) {
	b, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, apperr.NewNotFoundError("branch")
	}

	if err := b.Update(input); err != nil {
		return nil, err
	}

	return repo.Update(ctx, b)
}
