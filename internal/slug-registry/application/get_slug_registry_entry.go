package application

import (
	"context"

	"github.com/trvux/elc-go/internal/slug-registry/domain"
)

func GetSlugRegistryEntry(ctx context.Context, repo domain.SlugRegistryRepository, slug string) (*domain.SlugRegistryEntry, error) {
	return repo.GetBySlug(ctx, slug)
}
