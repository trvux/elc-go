package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func ListUsers(ctx context.Context, userRepo domain.UserRepository) ([]*domain.User, error) {
	users, err := userRepo.GetAll(ctx)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	return users, nil
}
