package application

import (
	"context"

	"github.com/trvux/elc-go/internal/ai/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// UpdateProviderInput is a partial update — nil means "leave unchanged",
// same convention as domain.UpdateProviderInput this wraps. APIKey nil
// leaves the stored key untouched.
type UpdateProviderInput struct {
	ID            string
	DisplayName   *string
	BaseURL       *string
	APIKey        *string
	PricingDocURL *string
	IsActive      *bool
}

func UpdateProvider(ctx context.Context, repo domain.ProviderRepository, input UpdateProviderInput) (*domain.Provider, error) {
	provider, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, apperr.NewNotFoundError("ai provider")
	}

	// APIKey is passed to both calls: domain.Provider.Update validates it
	// (rejects an explicit empty string) and stages it in memory, but the
	// actual encrypted write happens via repo.Update's separate newAPIKey
	// parameter — see ProviderRepository.Update's doc comment for why
	// GetByID-loaded entities can't carry the plaintext key themselves.
	if err := provider.Update(domain.UpdateProviderInput{
		DisplayName:   input.DisplayName,
		BaseURL:       input.BaseURL,
		APIKey:        input.APIKey,
		PricingDocURL: input.PricingDocURL,
		IsActive:      input.IsActive,
	}); err != nil {
		return nil, err
	}

	return repo.Update(ctx, provider, input.APIKey)
}
