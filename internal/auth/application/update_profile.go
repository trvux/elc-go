package application

import (
	"context"
	"strings"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type UpdateProfileInput struct {
	Name      *string
	Email     *string
	AvatarURL *string
}

// UpdateProfile lets an authenticated user edit their own display name,
// email, and avatar — the self-service counterpart to UpdateUser (which is
// an admin acting on *someone else's* role/status, and deliberately
// forbidden on one's own account). There is no rank check here: every
// account is always allowed to edit its own profile.
func UpdateProfile(
	ctx context.Context,
	userRepo domain.UserRepository,
	actor *domain.User,
	input UpdateProfileInput,
) (*domain.User, error) {
	name := actor.Name()
	if input.Name != nil {
		name = *input.Name
	}

	email := actor.Email()
	if input.Email != nil {
		email = strings.ToLower(strings.TrimSpace(*input.Email))
		if email != actor.Email() {
			existing, err := userRepo.GetByEmail(ctx, email)
			if err != nil {
				return nil, apperr.NewInternalError(err)
			}
			if existing != nil {
				return nil, apperr.NewConflictError("a user with this email already exists")
			}
		}
	}

	if err := actor.UpdateProfile(name, email); err != nil {
		return nil, err
	}

	if input.AvatarURL != nil {
		actor.SetAvatarURL(*input.AvatarURL)
	}

	updated, err := userRepo.Update(ctx, actor)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	return updated, nil
}
