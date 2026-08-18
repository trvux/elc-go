package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
)

type CreateModelInput struct {
	ProviderID       string
	ModelName        string
	DisplayName      string
	Role             domain.ModelRole
	Pricing          domain.Pricing
	FallbackPriority int
}

func CreateModel(ctx context.Context, repo domain.ModelRepository, input CreateModelInput) (*domain.Model, error) {
	model, err := domain.NewModel(input.ProviderID, input.ModelName, input.DisplayName, input.Role, input.Pricing, input.FallbackPriority)
	if err != nil {
		return nil, err
	}
	return repo.Create(ctx, model)
}
