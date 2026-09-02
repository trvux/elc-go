package application

import (
	"context"
	"crypto/subtle"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// magicLinkMaxAttempts locks a magic link's token out after this many wrong
// code guesses, well before its TTL naturally expires — a 6-digit code has
// only 10^6 combinations, so leaving the full TTL window open to guessing
// would make brute force realistic.
const magicLinkMaxAttempts = 5

// VerifyMagicLinkByToken redeems the raw token from a clicked magic link
// (never sent as a URL query param — see RequestMagicLink and the emailed
// link's URL fragment).
func VerifyMagicLinkByToken(
	ctx context.Context,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	tokenRepo domain.VerificationTokenRepository,
	issuer domain.TokenIssuer,
	adminEmails []string,
	rawToken, userAgent, ipAddress string,
) (*LoginResult, error) {
	token, err := tokenRepo.GetByHash(ctx, domain.TokenPurposeMagicLink, domain.HashToken(rawToken))
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	if token == nil || !token.IsUsable() {
		return nil, apperr.NewUnauthorizedError("magic link is invalid or has expired")
	}
	return completeMagicLinkLogin(ctx, userRepo, sessionRepo, tokenRepo, issuer, adminEmails, token, userAgent, ipAddress)
}

// VerifyMagicLinkByCode redeems the 6-digit code emailed alongside the link,
// entered manually. It compares against the latest active token for this
// email in application code (constant-time) rather than an exact-match
// query, so a wrong guess can be counted — and the token locked out after
// magicLinkMaxAttempts — instead of looking identical to "no such token".
func VerifyMagicLinkByCode(
	ctx context.Context,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	tokenRepo domain.VerificationTokenRepository,
	issuer domain.TokenIssuer,
	adminEmails []string,
	email, code, userAgent, ipAddress string,
) (*LoginResult, error) {
	token, err := tokenRepo.GetLatestActiveByEmail(ctx, domain.TokenPurposeMagicLink, email)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	if token == nil {
		return nil, apperr.NewUnauthorizedError("invalid or expired code")
	}

	if subtle.ConstantTimeCompare([]byte(token.Code()), []byte(code)) != 1 {
		attempts, err := tokenRepo.IncrementAttempts(ctx, token.ID())
		if err != nil {
			return nil, apperr.NewInternalError(err)
		}
		if attempts >= magicLinkMaxAttempts {
			if err := tokenRepo.MarkConsumed(ctx, token.ID()); err != nil {
				return nil, apperr.NewInternalError(err)
			}
		}
		return nil, apperr.NewUnauthorizedError("invalid or expired code")
	}

	return completeMagicLinkLogin(ctx, userRepo, sessionRepo, tokenRepo, issuer, adminEmails, token, userAgent, ipAddress)
}

func completeMagicLinkLogin(
	ctx context.Context,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	tokenRepo domain.VerificationTokenRepository,
	issuer domain.TokenIssuer,
	adminEmails []string,
	token *domain.VerificationToken,
	userAgent, ipAddress string,
) (*LoginResult, error) {
	if err := tokenRepo.MarkConsumed(ctx, token.ID()); err != nil {
		return nil, apperr.NewInternalError(err)
	}

	user, err := resolveOAuthUser(ctx, userRepo, token.Email(), nil, "", "", adminEmails)
	if err != nil {
		return nil, err
	}

	return finishLogin(ctx, userRepo, sessionRepo, issuer, user, userAgent, ipAddress)
}
