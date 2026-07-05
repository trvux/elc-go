package application

import "time"

// Token lifetimes. Kept short for access tokens (stateless, can't be revoked
// before expiry) and generous enough for invite/reset links that a real
// human needs to receive an email and click it.
const (
	InviteTokenTTL        = 72 * time.Hour
	PasswordResetTokenTTL = 30 * time.Minute
	RefreshTokenTTL       = 30 * 24 * time.Hour
)
