package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// ForgotPassword never reveals whether the email exists — the handler always
// responds with the same generic "check your email" message regardless of
// what this function does internally. Only a real infrastructure failure
// (DB/email down) should surface as an error.
func ForgotPassword(
	ctx context.Context,
	userRepo domain.UserRepository,
	tokenRepo domain.VerificationTokenRepository,
	emailSender domain.EmailSender,
	email string,
) error {
	user, err := userRepo.GetByEmail(ctx, email)
	if err != nil {
		return apperr.NewInternalError(err)
	}
	if user == nil || !user.IsActive() {
		return nil
	}

	token, raw, err := domain.NewPasswordResetToken(user.ID(), user.Email(), PasswordResetTokenTTL)
	if err != nil {
		return err
	}

	if _, err := tokenRepo.Create(ctx, token); err != nil {
		return apperr.NewInternalError(err)
	}

	if err := emailSender.SendPasswordReset(ctx, user.Email(), raw); err != nil {
		return apperr.NewInternalError(err)
	}

	return nil
}
