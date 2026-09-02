package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type UpdateUserInput struct {
	Role   *domain.Role
	Status *domain.UserStatus
}

// UpdateUser changes an existing user's role and/or status. actor must
// outrank (or match) the target's current role, and outrank (or match) any
// newly requested role — one guard blocks both "demote someone above you"
// and "promote someone above yourself". Nobody can act on their own account
// through this path, to rule out both self-lockout and self-escalation.
func UpdateUser(
	ctx context.Context,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	actor *domain.User,
	targetID string,
	input UpdateUserInput,
) (*domain.User, error) {
	if actor.ID() == targetID {
		return nil, apperr.NewForbiddenError("cannot manage your own account")
	}

	target, err := userRepo.GetByID(ctx, targetID)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	if target == nil {
		return nil, apperr.NewNotFoundError("user")
	}
	if !actor.CanManageUser(target) {
		return nil, apperr.NewForbiddenError("insufficient rank to manage this user")
	}

	if input.Role != nil {
		if !actor.CanGrantRole(*input.Role) {
			return nil, apperr.NewForbiddenError("cannot assign a role higher than your own")
		}
		target.SetRole(*input.Role)
	}

	disabling := false
	if input.Status != nil {
		switch *input.Status {
		case domain.UserStatusDisabled:
			target.Disable()
			disabling = true
		case domain.UserStatusActive:
			target.Activate()
		default:
			return nil, apperr.NewValidationError("validation failed", map[string][]string{"status": {"invalid status"}})
		}
	}

	updated, err := userRepo.Update(ctx, target)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}

	// Disabling an account should end its access immediately, not just at
	// its next access-token expiry.
	if disabling {
		if err := sessionRepo.RevokeAllByUserID(ctx, target.ID()); err != nil {
			return nil, apperr.NewInternalError(err)
		}
	}

	return updated, nil
}
