package application

import (
	"context"

	"github.com/trvux/elc-go/internal/contact/domain"
)

func DeleteContact(ctx context.Context, repo domain.ContactRepository, id string) error {
	return repo.Delete(ctx, id)
}
