package application

import "time"

// Token lifetimes. Kept short for access tokens (stateless, can't be revoked
// before expiry); MagicLinkTTL is short deliberately (see VerifyMagicLink's
// attempt-lockout — a longer window just gives a code-guessing attacker more
// time within the same rate-limit budget).
const (
	MagicLinkTTL    = 10 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)
