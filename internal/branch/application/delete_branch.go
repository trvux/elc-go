package application

import (
	"context"

	"github.com/trvux/elc-go/internal/branch/domain"
)

func DeleteBranch(ctx context.Context, repo domain.BranchRepository, id string) error {
	return repo.Delete(ctx, id)
}
