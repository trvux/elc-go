package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// RefreshToken exchanges a still-valid refresh token for a new access token,
// rotating the refresh token in the process (old one is revoked, a new one
// issued) so a stolen-and-replayed old token stops working the moment the
// legitimate client refreshes.
func RefreshToken(
	ctx context.Context,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	issuer domain.TokenIssuer,
	refreshToken string,
) (*LoginResult, error) {
	session, err := sessionRepo.GetByTokenHash(ctx, domain.HashToken(refreshToken))
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	if session == nil || !session.IsValid() {
		return nil, apperr.NewUnauthorizedError("session expired, please log in again")
	}

	user, err := userRepo.GetByID(ctx, session.UserID())
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	if user == nil || !user.IsActive() {
		return nil, apperr.NewUnauthorizedError("session expired, please log in again")
	}

	if err := sessionRepo.Revoke(ctx, session.ID()); err != nil {
		return nil, apperr.NewInternalError(err)
	}

	newSession, rawRefresh, err := domain.NewSession(user.ID(), session.UserAgent(), session.IPAddress(), RefreshTokenTTL)
	if err != nil {
		return nil, err
	}
	if _, err := sessionRepo.Create(ctx, newSession); err != nil {
		return nil, apperr.NewInternalError(err)
	}

	accessToken, err := issuer.IssueAccessToken(user.ID(), user.Role())
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}

	return &LoginResult{User: user, AccessToken: accessToken, RefreshToken: rawRefresh}, nil
}
