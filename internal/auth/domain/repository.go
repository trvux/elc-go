package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *User) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	// GetByIdentifier looks a user up by username OR email, so login can
	// accept either without the caller needing to know which one it is.
	GetByIdentifier(ctx context.Context, identifier string) (*User, error)
	ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error)
	// GetAll backs the admin-panel user management screen — the account list
	// is small (a handful of staff), so no pagination is needed yet.
	GetAll(ctx context.Context) ([]*User, error)
}

type VerificationTokenRepository interface {
	Create(ctx context.Context, token *VerificationToken) (*VerificationToken, error)
	GetByHash(ctx context.Context, purpose TokenPurpose, tokenHash string) (*VerificationToken, error)
	MarkConsumed(ctx context.Context, id string) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) (*Session, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
}

// PasswordHasher is implemented by infrastructure (bcrypt) so domain/
// application never import a crypto library directly.
type PasswordHasher interface {
	Hash(password string) (string, error)
	// Compare returns nil if password matches hash, an error otherwise.
	Compare(hash, password string) error
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
	SendInvite(ctx context.Context, to string, rawToken string, role Role) error
	SendPasswordReset(ctx context.Context, to string, rawToken string) error
}
