package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *User) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	// GetByGoogleSub looks a user up by their linked Google account — tried
	// before GetByEmail on Google login, so a user who changed their email on
	// Google still resolves to the same row.
	GetByGoogleSub(ctx context.Context, sub string) (*User, error)
	// GetAll backs the admin-panel user management screen — the account list
	// is small (a handful of staff/members), so no pagination is needed yet.
	GetAll(ctx context.Context) ([]*User, error)
}

type VerificationTokenRepository interface {
	Create(ctx context.Context, token *VerificationToken) (*VerificationToken, error)
	GetByHash(ctx context.Context, purpose TokenPurpose, tokenHash string) (*VerificationToken, error)
	// GetLatestActiveByEmail returns the newest not-yet-consumed, not-yet-
	// expired token for this email — used by magic-link code verification to
	// compare the code in application code (see VerifyMagicLink) instead of
	// an exact-match query, so wrong guesses can be counted and locked out.
	GetLatestActiveByEmail(ctx context.Context, purpose TokenPurpose, email string) (*VerificationToken, error)
	MarkConsumed(ctx context.Context, id string) error
	// IncrementAttempts records one more wrong code guess and returns the new
	// count, so the caller can lock the token out after too many.
	IncrementAttempts(ctx context.Context, id string) (int, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) (*Session, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
}

// TokenIssuer issues stateless access tokens (JWT). Verification happens on
// the HTTP middleware side, not through this interface.
type TokenIssuer interface {
	IssueAccessToken(userID string, role Role) (string, error)
}

// EmailSender is implemented by infrastructure. It receives the raw token,
// not a pre-built link — building the link (which needs the public base URL,
// a deployment concern) is infrastructure's job, not application's.
type EmailSender interface {
	SendMagicLink(ctx context.Context, to string, rawToken string, code string) error
}
