package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// RequestMagicLink creates a magic-link token for the given email and emails
// it — deliberately doesn't check whether an account already exists for this
// email (unlike the old forgot-password flow): a new email creates a
// RoleMember account the first time its link/code is actually redeemed (see
// VerifyMagicLink), so "does this account exist" isn't information worth
// hiding or revealing here, it just doesn't apply yet. Rate limiting against
// spamming an inbox lives in the handler (see presentation.AuthHandler), same
// as every other public endpoint in this module.
func RequestMagicLink(
	ctx context.Context,
	tokenRepo domain.VerificationTokenRepository,
	emailSender domain.EmailSender,
	email string,
) error {
	token, rawToken, code, err := domain.NewMagicLinkToken(email, MagicLinkTTL)
	if err != nil {
		return err
	}

	if _, err := tokenRepo.Create(ctx, token); err != nil {
		return apperr.NewInternalError(err)
	}

	if err := emailSender.SendMagicLink(ctx, token.Email(), rawToken, code); err != nil {
		return apperr.NewInternalError(err)
	}

	return nil
}
