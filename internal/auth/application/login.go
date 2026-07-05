package application

import (
	"context"
	"time"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type LoginInput struct {
	// Identifier is a username or an email — the caller doesn't say which.
	Identifier string
	Password   string
	UserAgent  string
	IPAddress  string
}

type LoginResult struct {
	User         *domain.User
	AccessToken  string
	RefreshToken string
}

func Login(
	ctx context.Context,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	hasher domain.PasswordHasher,
	issuer domain.TokenIssuer,
	input LoginInput,
) (*LoginResult, error) {
	user, err := userRepo.GetByIdentifier(ctx, input.Identifier)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	// Same generic error whether the identifier doesn't exist or the password
	// is wrong — never tell an attacker which half of the guess was right.
	if user == nil || !user.IsActive() {
		return nil, apperr.NewUnauthorizedError("invalid username/email or password")
	}
	if err := hasher.Compare(user.PasswordHash(), input.Password); err != nil {
		return nil, apperr.NewUnauthorizedError("invalid username/email or password")
	}

	accessToken, err := issuer.IssueAccessToken(user.ID(), user.Role())
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}

	session, rawRefresh, err := domain.NewSession(user.ID(), input.UserAgent, input.IPAddress, RefreshTokenTTL)
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
