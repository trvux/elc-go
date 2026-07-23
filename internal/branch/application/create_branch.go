package application

import (
	"context"

	"github.com/trvux/elc-go/internal/branch/domain"
)

func CreateBranch(ctx context.Context, repo domain.BranchRepository, input domain.CreateBranchInput) (*domain.Branch, error) {
	b, err := domain.NewBranch(
		input.Name,
		input.Slug,
		input.Address,
		input.Phone,
		input.Email,
		input.MapsURL,
		input.MapsEmbed,
		input.ProvinceCode,
		input.ProvinceName,
		input.WardCode,
		input.WardName,
		input.PostalCode,
		input.Description,
		input.Images,
		input.IsPublished,
		input.OrderIndex,
		input.MetaTitle,
		input.MetaDescription,
	)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, b)
}
