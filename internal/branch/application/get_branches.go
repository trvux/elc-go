package application

import (
	"context"

	"github.com/trvux/elc-go/internal/branch/domain"
)

func GetBranches(ctx context.Context, repo domain.BranchRepository, filter domain.BranchFilter) ([]*domain.Branch, error) {
	return repo.GetAll(ctx, filter)
}

func GetBranchByID(ctx context.Context, repo domain.BranchRepository, id string) (*domain.Branch, error) {
	return repo.GetByID(ctx, id)
}

func GetBranchBySlug(ctx context.Context, repo domain.BranchRepository, slug string) (*domain.Branch, error) {
	return repo.GetBySlug(ctx, slug)
}

func CountBranches(ctx context.Context, repo domain.BranchRepository, filter domain.BranchFilter) (int, error) {
	return repo.Count(ctx, filter)
}
