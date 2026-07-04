package application

import (
	"context"

	"github.com/trvux/elc-go/internal/contact/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func UpdateContact(ctx context.Context, repo domain.ContactRepository, input domain.UpdateContactInput) (*domain.Contact, error) {
	contact, err := repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if contact == nil {
		return nil, apperr.NewNotFoundError("contact")
	}

	if input.Type != nil {
		if err := contact.UpdateType(*input.Type); err != nil {
			return nil, err
		}
	}
	if input.Value != nil {
		if err := contact.UpdateValue(*input.Value); err != nil {
			return nil, err
		}
	}
	if input.Label != nil {
		if err := contact.UpdateLabel(input.Label); err != nil {
			return nil, err
		}
	}
	if input.IsActive != nil {
		contact.SetActive(*input.IsActive)
	}
	if input.OrderIndex != nil {
		contact.Reorder(*input.OrderIndex)
	}

	return repo.Update(ctx, contact)
}
