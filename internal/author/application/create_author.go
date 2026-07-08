package application

import (
	"context"

	"github.com/trvux/elc-go/internal/author/domain"
)

func CreateAuthor(ctx context.Context, repo domain.AuthorRepository, input domain.CreateAuthorInput) (*domain.Author, error) {
	author, err := domain.NewAuthor(input.Name, input.Slug, input.AvatarURL, input.Bio)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, author)
}
