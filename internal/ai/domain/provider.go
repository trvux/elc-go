package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Provider is one configured LLM vendor (DeepSeek, GLM, ...): the base URL
// and API key an OpenAI-compatible client needs to reach it. apiKey is held
// here as plaintext in memory only — infrastructure encrypts it before
// writing to Postgres and decrypts on read (see
// internal/ai/infrastructure/secret_crypto.go); this domain type never
// touches ciphertext.
type Provider struct {
	id            string
	name          string
	displayName   string
	baseURL       string
	apiKey        string
	pricingDocURL string
	isActive      bool
	createdAt     time.Time
	updatedAt     time.Time
}

func NewProvider(name, displayName, baseURL, apiKey, pricingDocURL string) (*Provider, error) {
	if errs := validateProviderName(name); len(errs) > 0 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
	}
	if errs := validateBaseURL(baseURL); len(errs) > 0 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"baseUrl": errs})
	}
	if errs := validateAPIKey(apiKey); len(errs) > 0 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"apiKey": errs})
	}
	return &Provider{
		name:          name,
		displayName:   displayName,
		baseURL:       baseURL,
		apiKey:        apiKey,
		pricingDocURL: pricingDocURL,
		isActive:      true,
	}, nil
}

func RehydrateProvider(id, name, displayName, baseURL, apiKey, pricingDocURL string, isActive bool, createdAt, updatedAt time.Time) *Provider {
	return &Provider{
		id: id, name: name, displayName: displayName, baseURL: baseURL,
		apiKey: apiKey, pricingDocURL: pricingDocURL, isActive: isActive,
		createdAt: createdAt, updatedAt: updatedAt,
	}
}

func (p *Provider) ID() string            { return p.id }
func (p *Provider) Name() string          { return p.name }
func (p *Provider) DisplayName() string   { return p.displayName }
func (p *Provider) BaseURL() string       { return p.baseURL }
func (p *Provider) APIKey() string        { return p.apiKey }
func (p *Provider) PricingDocURL() string { return p.pricingDocURL }
func (p *Provider) IsActive() bool        { return p.isActive }
func (p *Provider) CreatedAt() time.Time  { return p.createdAt }
func (p *Provider) UpdatedAt() time.Time  { return p.updatedAt }

// UpdateProviderInput is a partial update — nil means "leave unchanged".
// APIKey nil leaves the stored (encrypted) key untouched, so editing e.g.
// just DisplayName from the admin panel never requires re-pasting a secret.
type UpdateProviderInput struct {
	DisplayName   *string
	BaseURL       *string
	APIKey        *string
	PricingDocURL *string
	IsActive      *bool
}

func (p *Provider) Update(input UpdateProviderInput) error {
	changed := false

	if input.DisplayName != nil {
		p.displayName = *input.DisplayName
		changed = true
	}
	if input.BaseURL != nil {
		if errs := validateBaseURL(*input.BaseURL); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"baseUrl": errs})
		}
		p.baseURL = *input.BaseURL
		changed = true
	}
	if input.APIKey != nil {
		if errs := validateAPIKey(*input.APIKey); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"apiKey": errs})
		}
		p.apiKey = *input.APIKey
		changed = true
	}
	if input.PricingDocURL != nil {
		p.pricingDocURL = *input.PricingDocURL
		changed = true
	}
	if input.IsActive != nil {
		p.isActive = *input.IsActive
		changed = true
	}

	if changed {
		p.updatedAt = time.Now()
	}
	return nil
}

func validateProviderName(name string) []string {
	var errs []string
	if name == "" {
		errs = append(errs, "name cannot be empty")
	}
	return errs
}

func validateBaseURL(u string) []string {
	var errs []string
	if u == "" {
		errs = append(errs, "base url cannot be empty")
	}
	return errs
}

func validateAPIKey(k string) []string {
	var errs []string
	if k == "" {
		errs = append(errs, "api key cannot be empty")
	}
	return errs
}
