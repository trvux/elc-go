package application

import (
	"context"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type GoogleLoginInput struct {
	Code        string
	RedirectURI string
	UserAgent   string
	IPAddress   string
}

// GoogleLogin exchanges a Google authorization code for a verified identity,
// resolves it to a user (creating a RoleMember account on first sign-in —
// see resolveOAuthUser), and issues a session the same way every other login
// path does.
func GoogleLogin(
	ctx context.Context,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	issuer domain.TokenIssuer,
	googleAuth domain.GoogleAuthenticator,
	adminEmails []string,
	input GoogleLoginInput,
) (*LoginResult, error) {
	info, err := googleAuth.Exchange(ctx, input.Code, input.RedirectURI)
	if err != nil {
		return nil, apperr.NewUnauthorizedError("could not verify google account")
	}

	user, err := resolveOAuthUser(ctx, userRepo, info.Email, &info.Sub, info.Name, info.Picture, adminEmails)
	if err != nil {
		return nil, err
	}

	return finishLogin(ctx, userRepo, sessionRepo, issuer, user, input.UserAgent, input.IPAddress)
}
