package domain

import "context"

type Repository interface {
	List(ctx context.Context, filter Filter) (items []*Redirect, total int, err error)
	GetByID(ctx context.Context, id string) (*Redirect, error)
	// GetByFrom returns nil, nil when no redirect starts at fromPath.
	GetByFrom(ctx context.Context, fromPath string) (*Redirect, error)
	Create(ctx context.Context, r *Redirect) (*Redirect, error)
	Update(ctx context.Context, r *Redirect) (*Redirect, error)
	Delete(ctx context.Context, id string) error
	// Repoint rewrites every redirect whose destination is oldTarget so it
	// points at newTarget, which keeps the table free of redirect chains.
	Repoint(ctx context.Context, oldTarget, newTarget string) error
}
