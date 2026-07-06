package application

import (
	"context"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// RegisterZaloFollower upserts a staff Zalo account as an active
// notification recipient — called from the OA webhook handler whenever a
// `follow` or user-sends-a-message event arrives, so no one ever has to be
// added by hand.
func RegisterZaloFollower(ctx context.Context, repo domain.ZaloFollowerRepository, zaloUserID string, displayName *string) error {
	return repo.Upsert(ctx, domain.NewZaloOAFollower(zaloUserID, displayName))
}

// DeactivateZaloFollower is called on an `unfollow` event — Zalo will
// reject any further push to this user until they follow again.
func DeactivateZaloFollower(ctx context.Context, repo domain.ZaloFollowerRepository, zaloUserID string) error {
	return repo.SetActive(ctx, zaloUserID, false)
}
