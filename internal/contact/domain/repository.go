package domain

import "context"

type ContactRepository interface {
	GetAll(ctx context.Context, filter ContactFilter) ([]*Contact, error)
	Count(ctx context.Context, filter ContactFilter) (int, error)
	GetByID(ctx context.Context, id string) (*Contact, error)
	Create(ctx context.Context, contact *Contact) (*Contact, error)
	Update(ctx context.Context, contact *Contact) (*Contact, error)
	Delete(ctx context.Context, id string) error
}
