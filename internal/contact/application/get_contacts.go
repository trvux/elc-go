package application

import (
	"context"

	"github.com/trvux/elc-go/internal/contact/domain"
)

func GetContacts(ctx context.Context, repo domain.ContactRepository, filter domain.ContactFilter) ([]*domain.Contact, error) {
	return repo.GetAll(ctx, filter)
}

func GetContactByID(ctx context.Context, repo domain.ContactRepository, id string) (*domain.Contact, error) {
	return repo.GetByID(ctx, id)
}
