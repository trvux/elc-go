package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/branch/domain"
)

type fakeBranchRepository struct {
	items map[string]*domain.Branch
}

func newFakeBranchRepository() *fakeBranchRepository {
	return &fakeBranchRepository{items: map[string]*domain.Branch{}}
}

func (r *fakeBranchRepository) GetAll(ctx context.Context, filter domain.BranchFilter) ([]*domain.Branch, error) {
	result := make([]*domain.Branch, 0, len(r.items))
	for _, b := range r.items {
		if filter.IsPublished != nil && b.IsPublished() != *filter.IsPublished {
			continue
		}
		if filter.Search != "" &&
			!strings.Contains(strings.ToLower(b.Name()), strings.ToLower(filter.Search)) &&
			!strings.Contains(strings.ToLower(b.Address()), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, b)
	}
	return result, nil
}

func (r *fakeBranchRepository) Count(ctx context.Context, filter domain.BranchFilter) (int, error) {
	list, err := r.GetAll(ctx, filter)
	if err != nil {
		return 0, err
	}
	return len(list), nil
}

func (r *fakeBranchRepository) GetByID(ctx context.Context, id string) (*domain.Branch, error) {
	b, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	return b, nil
}

func (r *fakeBranchRepository) GetBySlug(ctx context.Context, slug string) (*domain.Branch, error) {
	for _, b := range r.items {
		if b.Slug() == slug {
			return b, nil
		}
	}
	return nil, nil
}

func (r *fakeBranchRepository) Create(ctx context.Context, branch *domain.Branch) (*domain.Branch, error) {
	id := fmt.Sprintf("id-%d", len(r.items)+1)
	now := time.Now()
	created := domain.RehydrateBranch(
		id, branch.Name(), branch.Slug(), branch.Address(), branch.Phone(), branch.Email(),
		branch.MapsURL(), branch.MapsEmbed(), branch.Description(), branch.Images(),
		branch.IsPublished(), branch.OrderIndex(), branch.MetaTitle(), branch.MetaDescription(),
		now, now, nil,
	)
	r.items[id] = created
	return created, nil
}

func (r *fakeBranchRepository) Update(ctx context.Context, branch *domain.Branch) (*domain.Branch, error) {
	r.items[branch.ID()] = branch
	return branch, nil
}

func (r *fakeBranchRepository) Delete(ctx context.Context, id string) error {
	delete(r.items, id)
	return nil
}
