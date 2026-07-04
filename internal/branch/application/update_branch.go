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

	if input.Name != nil {
		if err := b.UpdateName(*input.Name); err != nil {
			return nil, err
		}
	}
	if input.Slug != nil {
		if err := b.UpdateSlug(*input.Slug); err != nil {
			return nil, err
		}
	}
	if input.Address != nil {
		if err := b.UpdateAddress(*input.Address); err != nil {
			return nil, err
		}
	}
	if input.Phone != nil {
		if err := b.UpdatePhone(*input.Phone); err != nil {
			return nil, err
		}
	}
	if input.Email != nil {
		if err := b.UpdateEmail(*input.Email); err != nil {
			return nil, err
		}
	}
	if input.MapsURL != nil {
		if err := b.UpdateMapsURL(*input.MapsURL); err != nil {
			return nil, err
		}
	}
	if input.MapsEmbed != nil {
		if err := b.UpdateMapsEmbed(*input.MapsEmbed); err != nil {
			return nil, err
		}
	}
	if input.Description != nil {
		b.UpdateDescription(input.Description)
	}
	if input.ImageUrl != nil {
		b.UpdateImageUrl(input.ImageUrl)
	}
	if input.IsPublished != nil {
		b.SetPublished(*input.IsPublished)
	}
	if input.OrderIndex != nil {
		b.Reorder(*input.OrderIndex)
	}
	if input.MetaTitle != nil {
		b.UpdateMetaTitle(input.MetaTitle)
	}
	if input.MetaDescription != nil {
		b.UpdateMetaDescription(input.MetaDescription)
	}

	return repo.Update(ctx, b)
}
