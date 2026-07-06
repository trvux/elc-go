package domain

import "time"

// ZaloOAFollower is a staff member who has followed the company's Zalo OA
// and messaged it at least once — the only way Zalo allows the OA to push a
// message back (see ZaloOASender). Captured automatically via the OA
// webhook's follow/user_send_text events, never entered by hand.
type ZaloOAFollower struct {
	id          string
	zaloUserID  string
	displayName *string
	isActive    bool
	followedAt  time.Time
}

func NewZaloOAFollower(zaloUserID string, displayName *string) *ZaloOAFollower {
	return &ZaloOAFollower{
		zaloUserID:  zaloUserID,
		displayName: displayName,
		isActive:    true,
		followedAt:  time.Now(),
	}
}

func RehydrateZaloOAFollower(id, zaloUserID string, displayName *string, isActive bool, followedAt time.Time) *ZaloOAFollower {
	return &ZaloOAFollower{
		id:          id,
		zaloUserID:  zaloUserID,
		displayName: displayName,
		isActive:    isActive,
		followedAt:  followedAt,
	}
}

func (f *ZaloOAFollower) ID() string            { return f.id }
func (f *ZaloOAFollower) ZaloUserID() string    { return f.zaloUserID }
func (f *ZaloOAFollower) DisplayName() *string  { return f.displayName }
func (f *ZaloOAFollower) IsActive() bool        { return f.isActive }
func (f *ZaloOAFollower) FollowedAt() time.Time { return f.followedAt }

// ZaloOAToken is the single current OAuth token pair for the company's Zalo
// OA app. Zalo rotates the refresh token on every use (the old one becomes
// invalid), so both must be persisted together on every refresh — see
// ZaloTokenRepository.Save.
type ZaloOAToken struct {
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

func NewZaloOAToken(accessToken, refreshToken string, expiresAt time.Time) *ZaloOAToken {
	return &ZaloOAToken{accessToken: accessToken, refreshToken: refreshToken, expiresAt: expiresAt}
}

func (t *ZaloOAToken) AccessToken() string  { return t.accessToken }
func (t *ZaloOAToken) RefreshToken() string { return t.refreshToken }
func (t *ZaloOAToken) ExpiresAt() time.Time { return t.expiresAt }

// NeedsRefresh reports whether the access token is at or past the safety
// margin before its real expiry — access tokens live 1 hour, so refreshing
// with minutes to spare avoids ever sending a request with an expired one.
func (t *ZaloOAToken) NeedsRefresh(margin time.Duration) bool {
	return time.Now().Add(margin).After(t.expiresAt)
}
