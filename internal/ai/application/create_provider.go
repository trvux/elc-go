package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
)

type CreateProviderInput struct {
	Name          string
	DisplayName   string
	BaseURL       string
	APIKey        string
	PricingDocURL string
}

func CreateProvider(ctx context.Context, repo domain.ProviderRepository, input CreateProviderInput) (*domain.Provider, error) {
	provider, err := domain.NewProvider(input.Name, input.DisplayName, input.BaseURL, input.APIKey, input.PricingDocURL)
	if err != nil {
		return nil, err
	}
	return repo.Create(ctx, provider)
}
