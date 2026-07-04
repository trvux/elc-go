package application

import (
	"context"
	"fmt"

	"github.com/trvux/elc-go/internal/contact/domain"
)

// fakeContactRepository is an in-memory stand-in for PostgresContactRepository,
// used only in tests so the application layer can be tested without a real DB.
type fakeContactRepository struct {
	contacts map[string]*domain.Contact
}

func newFakeContactRepository() *fakeContactRepository {
	return &fakeContactRepository{contacts: map[string]*domain.Contact{}}
}

func (r *fakeContactRepository) GetAll(ctx context.Context, filter domain.ContactFilter) ([]*domain.Contact, error) {
	result := make([]*domain.Contact, 0, len(r.contacts))
	for _, c := range r.contacts {
		result = append(result, c)
	}
	return result, nil
}

func (r *fakeContactRepository) Count(ctx context.Context, filter domain.ContactFilter) (int, error) {
	return len(r.contacts), nil
}

func (r *fakeContactRepository) GetByID(ctx context.Context, id string) (*domain.Contact, error) {
	c, ok := r.contacts[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (r *fakeContactRepository) Create(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	id := fmt.Sprintf("id-%d", len(r.contacts)+1)
	created := domain.RehydrateContact(id, contact.Type(), contact.Label(), contact.Value(), contact.IsActive(), contact.OrderIndex())
	r.contacts[id] = created
	return created, nil
}

func (r *fakeContactRepository) Update(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	r.contacts[contact.ID()] = contact
	return contact, nil
}

func (r *fakeContactRepository) Delete(ctx context.Context, id string) error {
	delete(r.contacts, id)
	return nil
}
