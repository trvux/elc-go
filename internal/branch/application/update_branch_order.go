package application

import (
	"context"

	"github.com/trvux/elc-go/internal/branch/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdateBranchOrder(ctx context.Context, repo domain.BranchRepository, id string, orderIndex int) error {
	b, err := repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if b == nil {
		return apperr.NewNotFoundError("branch")
	}

	b.Reorder(orderIndex)
	_, err = repo.Update(ctx, b)
	return err
}
