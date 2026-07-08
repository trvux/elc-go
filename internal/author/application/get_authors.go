package application

import (
	"context"

	"github.com/trvux/elc-go/internal/author/domain"
)

func GetAuthors(ctx context.Context, repo domain.AuthorRepository, filter domain.AuthorFilter) ([]*domain.Author, error) {
	return repo.GetAll(ctx, filter)
}

func GetAuthorByID(ctx context.Context, repo domain.AuthorRepository, id string) (*domain.Author, error) {
	return repo.GetByID(ctx, id)
}

func GetAuthorBySlug(ctx context.Context, repo domain.AuthorRepository, slug string) (*domain.Author, error) {
	return repo.GetBySlug(ctx, slug)
}
