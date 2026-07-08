package application

import (
	"context"

	"github.com/trvux/elc-go/internal/author/domain"
)

func DeleteAuthor(ctx context.Context, repo domain.AuthorRepository, id string) error {
	return repo.SoftDelete(ctx, id)
}

func RestoreAuthor(ctx context.Context, repo domain.AuthorRepository, id string) error {
	return repo.Restore(ctx, id)
}
