package domain

import "context"

type SlugRegistryRepository interface {
	GetBySlug(ctx context.Context, slug string) (*SlugRegistryEntry, error)
}
