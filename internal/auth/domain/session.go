package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Session backs a refresh token. Access tokens (JWT) are stateless and never
// stored; refresh tokens are stored (hashed) so logout and password-reset can
// actually revoke them instead of just waiting out the access-token TTL.
type Session struct {
	id        string
	userID    string
	tokenHash string
	userAgent string
	ipAddress string
	expiresAt time.Time
	revokedAt *time.Time
	createdAt time.Time
}

// NewSession creates a new refresh-token session. Returns the raw token (to
// be set as an httpOnly cookie) alongside the entity.
func NewSession(userID, userAgent, ipAddress string, ttl time.Duration) (*Session, string, error) {
	if userID == "" {
		return nil, "", apperr.NewValidationError("validation failed", map[string][]string{"user_id": {"user_id is required"}})
	}

	raw, hash, err := GenerateSecureToken()
	if err != nil {
		return nil, "", apperr.NewInternalError(err)
	}

	return &Session{
		userID:    userID,
		tokenHash: hash,
		userAgent: userAgent,
		ipAddress: ipAddress,
		expiresAt: time.Now().Add(ttl),
	}, raw, nil
}

// RehydrateSession reconstructs a Session from a trusted DB row.
func RehydrateSession(
	id, userID, tokenHash, userAgent, ipAddress string,
	expiresAt time.Time,
	revokedAt *time.Time,
	createdAt time.Time,
) *Session {
	return &Session{
		id:        id,
		userID:    userID,
		tokenHash: tokenHash,
		userAgent: userAgent,
		ipAddress: ipAddress,
		expiresAt: expiresAt,
		revokedAt: revokedAt,
		createdAt: createdAt,
	}
}

func (s *Session) ID() string           { return s.id }
func (s *Session) UserID() string       { return s.userID }
func (s *Session) TokenHash() string    { return s.tokenHash }
func (s *Session) UserAgent() string    { return s.userAgent }
func (s *Session) IPAddress() string    { return s.ipAddress }
func (s *Session) ExpiresAt() time.Time { return s.expiresAt }
func (s *Session) CreatedAt() time.Time { return s.createdAt }

func (s *Session) IsValid() bool {
	return s.revokedAt == nil && time.Now().Before(s.expiresAt)
}

func (s *Session) Revoke(at time.Time) {
	s.revokedAt = &at
}
