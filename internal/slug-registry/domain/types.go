package domain

// SlugRegistryEntry maps a public-facing slug to the entity type/id it
// currently points to. Rows are written by DB triggers owned by each
// entity's own module (products, projects, categories, brands, groups,
// project_type) to enforce global slug uniqueness across all of them —
// this module only ever reads the shared table, never writes to it.
type SlugRegistryEntry struct {
	entityType string
	entityID   string
}

// RehydrateSlugRegistryEntry reconstructs from a trusted DB row — no
// validation. Only the infrastructure layer should call this.
func RehydrateSlugRegistryEntry(entityType, entityID string) *SlugRegistryEntry {
	return &SlugRegistryEntry{entityType: entityType, entityID: entityID}
}

func (e *SlugRegistryEntry) EntityType() string { return e.entityType }
func (e *SlugRegistryEntry) EntityID() string   { return e.entityID }
