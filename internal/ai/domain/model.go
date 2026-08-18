package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// ModelRole is which duty a Model can serve. 'chat' models answer
// customers, tried in fallback_priority order by SendChatMessage; a
// 'classifier' model runs the cheap pre-completion guardrail check.
type ModelRole string

const (
	ModelRoleChat       ModelRole = "chat"
	ModelRoleClassifier ModelRole = "classifier"
)

func (r ModelRole) IsValid() bool {
	return r == ModelRoleChat || r == ModelRoleClassifier
}

// Model is one provider's model exposed to this system, with its own
// pricing and fallback ordering.
type Model struct {
	id               string
	providerID       string
	modelName        string
	displayName      string
	role             ModelRole
	pricing          Pricing
	fallbackPriority int
	isDefault        bool
	isActive         bool
	createdAt        time.Time
	updatedAt        time.Time
}

func NewModel(providerID, modelName, displayName string, role ModelRole, pricing Pricing, fallbackPriority int) (*Model, error) {
	if providerID == "" {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"providerId": {"provider id cannot be empty"}})
	}
	if modelName == "" {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"modelName": {"model name cannot be empty"}})
	}
	if !role.IsValid() {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"role": {"role must be \"chat\" or \"classifier\""}})
	}
	return &Model{
		providerID:       providerID,
		modelName:        modelName,
		displayName:      displayName,
		role:             role,
		pricing:          pricing,
		fallbackPriority: fallbackPriority,
		isActive:         true,
	}, nil
}

func RehydrateModel(id, providerID, modelName, displayName string, role ModelRole, pricing Pricing, fallbackPriority int, isDefault, isActive bool, createdAt, updatedAt time.Time) *Model {
	return &Model{
		id: id, providerID: providerID, modelName: modelName, displayName: displayName,
		role: role, pricing: pricing, fallbackPriority: fallbackPriority,
		isDefault: isDefault, isActive: isActive, createdAt: createdAt, updatedAt: updatedAt,
	}
}

func (m *Model) ID() string            { return m.id }
func (m *Model) ProviderID() string    { return m.providerID }
func (m *Model) ModelName() string     { return m.modelName }
func (m *Model) DisplayName() string   { return m.displayName }
func (m *Model) Role() ModelRole       { return m.role }
func (m *Model) Pricing() Pricing      { return m.pricing }
func (m *Model) FallbackPriority() int { return m.fallbackPriority }
func (m *Model) IsDefault() bool       { return m.isDefault }
func (m *Model) IsActive() bool        { return m.isActive }
func (m *Model) CreatedAt() time.Time  { return m.createdAt }
func (m *Model) UpdatedAt() time.Time  { return m.updatedAt }

// UpdateModelInput is a partial update — nil means "leave unchanged".
// ProviderID/ModelName/Role are not updatable: changing which provider or
// wire model a row points to is a new row, not an edit, same as this
// codebase never lets Update change a Product's category via an implicit
// side channel.
type UpdateModelInput struct {
	DisplayName      *string
	Pricing          *Pricing
	FallbackPriority *int
	IsDefault        *bool
	IsActive         *bool
}

func (m *Model) Update(input UpdateModelInput) error {
	changed := false

	if input.DisplayName != nil {
		m.displayName = *input.DisplayName
		changed = true
	}
	if input.Pricing != nil {
		m.pricing = *input.Pricing
		changed = true
	}
	if input.FallbackPriority != nil {
		m.fallbackPriority = *input.FallbackPriority
		changed = true
	}
	if input.IsDefault != nil {
		m.isDefault = *input.IsDefault
		changed = true
	}
	if input.IsActive != nil {
		m.isActive = *input.IsActive
		changed = true
	}

	if changed {
		m.updatedAt = time.Now()
	}
	return nil
}

// ModelConfig is the resolved, runtime-ready view SendChatMessage's
// fallback loop iterates over: a Model joined with its Provider's base URL
// and decrypted API key. Never round-tripped to presentation — building one
// requires the decrypted key, which only ModelRepository.
// ListActiveConfigsByRole produces.
type ModelConfig struct {
	ProviderID       string
	ProviderName     string
	BaseURL          string
	APIKey           string
	ModelID          string
	ModelName        string
	Role             ModelRole
	Pricing          Pricing
	FallbackPriority int
}
