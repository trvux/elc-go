package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// ChangePassword lets an authenticated user change their own password by
// proving they know the current one (unlike ResetPassword, which proves
// identity via an emailed token instead — there is no such token here). Like
// ResetPassword, it revokes every session on success, including the one
// making this request: a password change should force re-login everywhere,
// and the codebase has no per-session "this is me" identifier to carve out
// an exception (access tokens are stateless JWTs, not tied to a session
// row), so this matches ResetPassword's existing precedent rather than
// inventing a new partial-revoke mechanism.
func ChangePassword(
	ctx context.Context,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	hasher domain.PasswordHasher,
	actor *domain.User,
	currentPassword, newPassword string,
) error {
	if err := hasher.Compare(actor.PasswordHash(), currentPassword); err != nil {
		return apperr.NewUnauthorizedError("current password is incorrect")
	}

	if errs := domain.ValidatePassword(newPassword); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"password": errs})
	}

	passwordHash, err := hasher.Hash(newPassword)
	if err != nil {
		return apperr.NewInternalError(err)
	}
	actor.SetPasswordHash(passwordHash)

	if _, err := userRepo.Update(ctx, actor); err != nil {
		return apperr.NewInternalError(err)
	}
	if err := sessionRepo.RevokeAllByUserID(ctx, actor.ID()); err != nil {
		return apperr.NewInternalError(err)
	}

	return nil
}
