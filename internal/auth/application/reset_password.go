package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// ResetPassword also revokes every existing session for the user — a
// password reset should force re-login everywhere, including on a device an
// attacker may still be logged into.
func ResetPassword(
	ctx context.Context,
	userRepo domain.UserRepository,
	tokenRepo domain.VerificationTokenRepository,
	sessionRepo domain.SessionRepository,
	hasher domain.PasswordHasher,
	rawToken string,
	newPassword string,
) error {
	token, err := tokenRepo.GetByHash(ctx, domain.TokenPurposePasswordReset, domain.HashToken(rawToken))
	if err != nil {
		return apperr.NewInternalError(err)
	}
	if token == nil || !token.IsUsable() {
		return apperr.NewUnauthorizedError("reset link is invalid or has expired")
	}

	if errs := domain.ValidatePassword(newPassword); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"password": errs})
	}

	user, err := userRepo.GetByID(ctx, token.UserID())
	if err != nil {
		return apperr.NewInternalError(err)
	}
	if user == nil {
		return apperr.NewUnauthorizedError("reset link is invalid or has expired")
	}

	passwordHash, err := hasher.Hash(newPassword)
	if err != nil {
		return apperr.NewInternalError(err)
	}
	user.SetPasswordHash(passwordHash)

	if _, err := userRepo.Update(ctx, user); err != nil {
		return apperr.NewInternalError(err)
	}
	if err := tokenRepo.MarkConsumed(ctx, token.ID()); err != nil {
		return apperr.NewInternalError(err)
	}
	if err := sessionRepo.RevokeAllByUserID(ctx, user.ID()); err != nil {
		return apperr.NewInternalError(err)
	}

	return nil
}
