package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// UpdateModelInput is a partial update — nil means "leave unchanged".
type UpdateModelInput struct {
	ID               string
	DisplayName      *string
	Pricing          *domain.Pricing
	FallbackPriority *int
	IsDefault        *bool
	IsActive         *bool
}

func UpdateModel(ctx context.Context, repo domain.ModelRepository, input UpdateModelInput) (*domain.Model, error) {
	model, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, apperr.NewNotFoundError("ai model")
	}

	if err := model.Update(domain.UpdateModelInput{
		DisplayName:      input.DisplayName,
		Pricing:          input.Pricing,
		FallbackPriority: input.FallbackPriority,
		IsDefault:        input.IsDefault,
		IsActive:         input.IsActive,
	}); err != nil {
		return nil, err
	}

	return repo.Update(ctx, model)
}
