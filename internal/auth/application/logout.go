package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Logout revokes the session behind the given raw refresh token. Idempotent:
// an already-revoked or unknown token is not an error, since the end state
// the caller wants ("I am logged out") is already true.
func Logout(ctx context.Context, sessionRepo domain.SessionRepository, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	session, err := sessionRepo.GetByTokenHash(ctx, domain.HashToken(refreshToken))
	if err != nil {
		return apperr.NewInternalError(err)
	}
	if session == nil {
		return nil
	}

	if err := sessionRepo.Revoke(ctx, session.ID()); err != nil {
		return apperr.NewInternalError(err)
	}
	return nil
}
