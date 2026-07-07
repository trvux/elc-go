package presentation

import (
	"github.com/trvux/elc-go/internal/slug-registry/domain"
)

type SlugRegistryEntryDTO struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

func toSlugRegistryEntryDTO(e *domain.SlugRegistryEntry) SlugRegistryEntryDTO {
	return SlugRegistryEntryDTO{
		EntityType: e.EntityType(),
		EntityID:   e.EntityID(),
	}
}
