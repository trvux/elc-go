package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func GetCurrentUser(ctx context.Context, userRepo domain.UserRepository, userID string) (*domain.User, error) {
	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	return user, nil
}
