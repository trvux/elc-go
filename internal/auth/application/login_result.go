package application

import (
	"context"
	"time"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// LoginResult is what every login path (Google, magic link, refresh) hands
// back to the presentation layer.
type LoginResult struct {
	User         *domain.User
	AccessToken  string
	RefreshToken string
}

// finishLogin issues a fresh session + access token for an already-resolved
// user — the shared tail of every login path, so session/JWT issuance and
// the last-login timestamp update live in one place instead of being
// duplicated per login method.
func finishLogin(
	ctx context.Context,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	issuer domain.TokenIssuer,
	user *domain.User,
	userAgent, ipAddress string,
) (*LoginResult, error) {
	if !user.IsActive() {
		return nil, apperr.NewUnauthorizedError("account is disabled")
	}

	accessToken, err := issuer.IssueAccessToken(user.ID(), user.Role())
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}

	session, rawRefresh, err := domain.NewSession(user.ID(), userAgent, ipAddress, RefreshTokenTTL)
	if err != nil {
		return nil, err
	}
	if _, err := sessionRepo.Create(ctx, session); err != nil {
		return nil, apperr.NewInternalError(err)
	}

	user.RecordLogin(time.Now())
	if _, err := userRepo.Update(ctx, user); err != nil {
		return nil, apperr.NewInternalError(err)
	}

	return &LoginResult{User: user, AccessToken: accessToken, RefreshToken: rawRefresh}, nil
}
