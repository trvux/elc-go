package domain

import (
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

type TokenPurpose string

const (
	TokenPurposeInvite        TokenPurpose = "invite"
	TokenPurposePasswordReset TokenPurpose = "password_reset"
)

// VerificationToken backs both the admin-invite flow and the forgot-password
// flow: a single-use, short-lived, high-entropy token whose hash (never the
// raw value) is stored in the DB. Purpose discriminates which fields apply —
// Role/InvitedBy are only meaningful for TokenPurposeInvite, UserID only for
// TokenPurposePasswordReset.
type VerificationToken struct {
	id         string
	purpose    TokenPurpose
	tokenHash  string
	email      string
	role       Role
	invitedBy  string
	userID     string
	expiresAt  time.Time
	consumedAt *time.Time
	createdAt  time.Time
}

// NewInviteToken creates an invite token for the given email/role. Returns
// the raw token (to be emailed, never stored) alongside the entity.
func NewInviteToken(email string, role Role, invitedBy string, ttl time.Duration) (*VerificationToken, string, error) {
	fields := map[string][]string{}

	email = strings.ToLower(strings.TrimSpace(email))
	if errs := validateEmail(email); len(errs) > 0 {
		fields["email"] = errs
	}
	if !role.IsValid() {
		fields["role"] = []string{"invalid role"}
	}
	if invitedBy == "" {
		fields["invited_by"] = []string{"invited_by is required"}
	}
	if len(fields) > 0 {
		return nil, "", apperr.NewValidationError("validation failed", fields)
	}

	raw, hash, err := GenerateSecureToken()
	if err != nil {
		return nil, "", apperr.NewInternalError(err)
	}

	return &VerificationToken{
		purpose:   TokenPurposeInvite,
		tokenHash: hash,
		email:     email,
		role:      role,
		invitedBy: invitedBy,
		expiresAt: time.Now().Add(ttl),
	}, raw, nil
}

// NewPasswordResetToken creates a password-reset token for an existing user.
func NewPasswordResetToken(userID, email string, ttl time.Duration) (*VerificationToken, string, error) {
	if userID == "" {
		return nil, "", apperr.NewValidationError("validation failed", map[string][]string{"user_id": {"user_id is required"}})
	}

	raw, hash, err := GenerateSecureToken()
	if err != nil {
		return nil, "", apperr.NewInternalError(err)
	}

	return &VerificationToken{
		purpose:   TokenPurposePasswordReset,
		tokenHash: hash,
		email:     strings.ToLower(strings.TrimSpace(email)),
		userID:    userID,
		expiresAt: time.Now().Add(ttl),
	}, raw, nil
}

// RehydrateVerificationToken reconstructs a token from a trusted DB row.
func RehydrateVerificationToken(
	id string,
	purpose TokenPurpose,
	tokenHash, email string,
	role Role,
	invitedBy, userID string,
	expiresAt time.Time,
	consumedAt *time.Time,
	createdAt time.Time,
) *VerificationToken {
	return &VerificationToken{
		id:         id,
		purpose:    purpose,
		tokenHash:  tokenHash,
		email:      email,
		role:       role,
		invitedBy:  invitedBy,
		userID:     userID,
		expiresAt:  expiresAt,
		consumedAt: consumedAt,
		createdAt:  createdAt,
	}
}

func (t *VerificationToken) ID() string            { return t.id }
func (t *VerificationToken) Purpose() TokenPurpose { return t.purpose }
func (t *VerificationToken) TokenHash() string     { return t.tokenHash }
func (t *VerificationToken) Email() string         { return t.email }
func (t *VerificationToken) Role() Role            { return t.role }
func (t *VerificationToken) InvitedBy() string     { return t.invitedBy }
func (t *VerificationToken) UserID() string        { return t.userID }
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
