package application

import (
	"context"

	"github.com/trvux/elc-go/internal/group/domain"
)

func CreateGroup(ctx context.Context, repo domain.GroupRepository, input domain.CreateGroupInput) (*domain.Group, error) {
	g, err := domain.NewGroup(
		input.Name,
		input.Slug,
		input.ImageURL,
		input.MetaTitle,
		input.MetaDescription,
		input.IsFeatured,
		input.OrderIndex,
		input.Content,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, g)
}
