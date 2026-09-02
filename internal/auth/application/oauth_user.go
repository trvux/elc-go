package application

import (
	"context"
	"slices"
	"strings"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

// resolveOAuthUser finds or creates the user behind an email proven by
// Google or magic link. googleSub is set only for a Google login — nil for
// magic link. Role is never taken from the caller: every new account is
// RoleMember, except an ADMIN_EMAILS allowlist match (checked only at
// creation time — an existing account's role is only ever changed
// afterward by an admin via UpdateUser, never silently re-derived from
// config on a later login).
func resolveOAuthUser(
	ctx context.Context,
	userRepo domain.UserRepository,
	email string,
	googleSub *string,
	name, avatarURL string,
	adminEmails []string,
) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if googleSub != nil {
		user, err := userRepo.GetByGoogleSub(ctx, *googleSub)
		if err != nil {
			return nil, apperr.NewInternalError(err)
		}
		if user != nil {
			return syncGoogleProfile(ctx, userRepo, user, name, avatarURL)
		}
	}

	user, err := userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	if user != nil {
		// An account created via magic link (or invited with a password)
		// signing in with Google for the first time — link it instead of
		// bouncing off the email's unique constraint trying to create a
		// second row for the same address.
		if googleSub != nil {
			if user.GoogleSub() == nil {
				user.SetGoogleSub(*googleSub)
			}
			return syncGoogleProfile(ctx, userRepo, user, name, avatarURL)
		}
		return user, nil
	}

	role := domain.RoleMember
	if slices.ContainsFunc(adminEmails, func(e string) bool { return strings.EqualFold(e, email) }) {
		role = domain.RoleSuperAdmin
	}

	newUser, err := domain.NewOAuthUser(email, name, avatarURL, googleSub, role)
	if err != nil {
		return nil, err
	}
	created, err := userRepo.Create(ctx, newUser)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	return created, nil
}

// syncGoogleProfile refreshes an existing user's avatar from Google on every
// Google login (a profile picture can change, and an account bootstrapped
// before Google login existed — e.g. via the old invite flow — never had one
// at all), and backfills name only if it's currently empty. Name is not
// unconditionally overwritten like avatar: unlike a profile picture, a
// display name can be deliberately customized via UpdateProfile, and
// clobbering that on every login would be surprising.
func syncGoogleProfile(ctx context.Context, userRepo domain.UserRepository, user *domain.User, name, avatarURL string) (*domain.User, error) {
	changed := false
	if avatarURL != "" && user.AvatarURL() != avatarURL {
		user.SetAvatarURL(avatarURL)
		changed = true
	}
	if name != "" && user.Name() == "" {
		user.SetName(name)
		changed = true
	}
	if !changed {
		return user, nil
	}

	updated, err := userRepo.Update(ctx, user)
	if err != nil {
		return nil, apperr.NewInternalError(err)
	}
	return updated, nil
}
