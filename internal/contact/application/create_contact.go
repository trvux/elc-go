package application

import (
	"context"

	"github.com/trvux/elc-go/internal/contact/domain"
)

func CreateContact(ctx context.Context, repo domain.ContactRepository, input domain.CreateContactInput) (*domain.Contact, error) {
	contact, err := domain.NewContact(input.Type, input.Label, input.Value, input.IsActive, input.OrderIndex)
	if err != nil {
		return nil, err
	}

	return repo.Create(ctx, contact)
}
