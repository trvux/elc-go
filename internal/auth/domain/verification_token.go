package domain

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

type TokenPurpose string

const TokenPurposeMagicLink TokenPurpose = "magic_link"

// VerificationToken backs the magic-link login flow: a single-use, short-
// lived link+code pair. The raw token (from the emailed link) and Code (the
// 6-digit number emailed alongside it, for manual entry) are two independent
// ways to redeem the same row — either satisfies it. Only the token's hash is
// ever stored, never the raw value; Code is stored as-is since knowing it is
// exactly what redeeming-by-code proves. Attempts counts wrong code guesses,
// so VerifyMagicLink can lock the row out before its TTL naturally expires.
type VerificationToken struct {
	id         string
	purpose    TokenPurpose
	tokenHash  string
	email      string
	code       string
	attempts   int
	expiresAt  time.Time
	consumedAt *time.Time
	createdAt  time.Time
}

// NewMagicLinkToken creates a magic-link token for the given email. Returns
// the entity, the raw token (for the emailed link, never stored), and the
// 6-digit code (emailed alongside it, for manual entry).
func NewMagicLinkToken(email string, ttl time.Duration) (token *VerificationToken, rawToken string, code string, err error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if errs := validateEmail(email); len(errs) > 0 {
		return nil, "", "", apperr.NewValidationError("validation failed", map[string][]string{"email": errs})
	}

	rawToken, hash, err := GenerateSecureToken()
	if err != nil {
		return nil, "", "", apperr.NewInternalError(err)
	}
	code, err = generateCode()
	if err != nil {
		return nil, "", "", apperr.NewInternalError(err)
	}

	return &VerificationToken{
		purpose:   TokenPurposeMagicLink,
		tokenHash: hash,
		email:     email,
		code:      code,
		expiresAt: time.Now().Add(ttl),
	}, rawToken, code, nil
}

// generateCode returns a random 6-digit code, zero-padded (e.g. "004213").
func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// RehydrateVerificationToken reconstructs a token from a trusted DB row.
func RehydrateVerificationToken(
	id string,
	purpose TokenPurpose,
	tokenHash, email, code string,
	attempts int,
	expiresAt time.Time,
	consumedAt *time.Time,
	createdAt time.Time,
) *VerificationToken {
	return &VerificationToken{
		id:         id,
		purpose:    purpose,
		tokenHash:  tokenHash,
		email:      email,
		code:       code,
		attempts:   attempts,
		expiresAt:  expiresAt,
		consumedAt: consumedAt,
		createdAt:  createdAt,
	}
}

func (t *VerificationToken) ID() string            { return t.id }
func (t *VerificationToken) Purpose() TokenPurpose { return t.purpose }
func (t *VerificationToken) TokenHash() string     { return t.tokenHash }
func (t *VerificationToken) Email() string         { return t.email }
func (t *VerificationToken) Code() string          { return t.code }
func (t *VerificationToken) Attempts() int         { return t.attempts }
func (t *VerificationToken) ExpiresAt() time.Time  { return t.expiresAt }
func (t *VerificationToken) CreatedAt() time.Time  { return t.createdAt }

func (t *VerificationToken) IsExpired() bool {
	return time.Now().After(t.expiresAt)
}

func (t *VerificationToken) IsConsumed() bool {
	return t.consumedAt != nil
}

// IsUsable is the single check every use case should call before honoring a
// token — expired or already-consumed tokens must be rejected identically
// (a generic "invalid or expired" error), never distinguished for the client.
func (t *VerificationToken) IsUsable() bool {
	return !t.IsExpired() && !t.IsConsumed()
}
