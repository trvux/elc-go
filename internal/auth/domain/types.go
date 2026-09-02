package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._]+$`)

type User struct {
	id           string
	username     string
	email        string
	passwordHash string
	name         string
	phone        string
	avatarURL    string
	googleSub    *string
	role         Role
	status       UserStatus
	lastLoginAt  *time.Time
	createdAt    time.Time
	updatedAt    time.Time
}

// NewUser validates input and creates a new User. This is the only
// constructor that creates a user, and it is only ever called from the
// accept-invite use case — there is no public self-registration path, so a
// stranger can never reach this by hitting an endpoint directly. name and
// phone are optional display/contact info, self-reported by the invitee —
// they are never used to authenticate or authorize.
func NewUser(username, email, passwordHash, name, phone string, role Role) (*User, error) {
	fields := map[string][]string{}

	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)

	if errs := validateUsername(username); len(errs) > 0 {
		fields["username"] = errs
	}
	if errs := validateEmail(email); len(errs) > 0 {
		fields["email"] = errs
	}
	if passwordHash == "" {
		fields["password"] = []string{"password hash is required"}
	}
	if !role.IsValid() {
		fields["role"] = []string{"invalid role"}
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	return &User{
		username:     username,
		email:        email,
		passwordHash: passwordHash,
		name:         name,
		phone:        phone,
		role:         role,
		status:       UserStatusActive,
	}, nil
}

// NewOAuthUser creates a user authenticated via Google or magic link —
// proven by owning the email inbox (or a verified Google account) instead of
// a password, so there is no passwordHash and no username to choose (see the
// migration dropping users.username's NOT NULL). googleSub is nil for a
// magic-link-only account. Unlike NewUser (only ever reachable via an
// existing admin's invite), this is the automatic account-creation path any
// email can trigger — role must therefore never be decided by the caller
// blindly; see application.resolveOAuthUser for the RoleMember-by-default,
// ADMIN_EMAILS-allowlist-exception logic that picks it.
func NewOAuthUser(email, name, avatarURL string, googleSub *string, role Role) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)

	if errs := validateEmail(email); len(errs) > 0 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"email": errs})
	}
	if !role.IsValid() {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"role": {"invalid role"}})
	}

	return &User{
		email:     email,
		name:      name,
		avatarURL: avatarURL,
		googleSub: googleSub,
		role:      role,
		status:    UserStatusActive,
	}, nil
}

// RehydrateUser reconstructs a User from a trusted DB row. No validation —
// only the infrastructure layer should call this.
func RehydrateUser(
	id, username, email, passwordHash, name, phone, avatarURL string,
	googleSub *string,
	role Role,
	status UserStatus,
	lastLoginAt *time.Time,
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:           id,
		username:     username,
		email:        email,
		passwordHash: passwordHash,
		name:         name,
		phone:        phone,
		avatarURL:    avatarURL,
		googleSub:    googleSub,
		role:         role,
		status:       status,
		lastLoginAt:  lastLoginAt,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

func (u *User) ID() string              { return u.id }
func (u *User) Username() string        { return u.username }
func (u *User) Email() string           { return u.email }
func (u *User) PasswordHash() string    { return u.passwordHash }
func (u *User) Name() string            { return u.name }
func (u *User) Phone() string           { return u.phone }
func (u *User) AvatarURL() string       { return u.avatarURL }
func (u *User) GoogleSub() *string      { return u.googleSub }
func (u *User) Role() Role              { return u.role }
func (u *User) Status() UserStatus      { return u.status }
func (u *User) LastLoginAt() *time.Time { return u.lastLoginAt }
func (u *User) CreatedAt() time.Time    { return u.createdAt }
func (u *User) UpdatedAt() time.Time    { return u.updatedAt }
func (u *User) IsActive() bool          { return u.status == UserStatusActive }

// CanGrantRole enforces the privilege-escalation guard: nobody can grant a
// role ranked higher than their own (see roleRank), and RoleUser/RoleMember
// cannot grant any role at all — granting admin-panel access is account
// management, not content work, and members have no admin-panel standing to
// begin with.
func (u *User) CanGrantRole(targetRole Role) bool {
	if !targetRole.IsValid() || u.role == RoleUser || u.role == RoleMember {
		return false
	}
	return roleRank[u.role] >= roleRank[targetRole]
}

// CanManageUser enforces the same privilege boundary as CanGrantRole but for
// an *existing* account: nobody can change the role/status of a user who
// currently outranks them. Combine with CanGrantRole(newRole) when the action
// is specifically a role change, so neither the target's current rank nor
// the requested new rank can exceed the actor's own.
func (u *User) CanManageUser(target *User) bool {
	if target == nil {
		return false
	}
	return roleRank[u.role] >= roleRank[target.role]
}

func (u *User) SetPasswordHash(hash string) {
	u.passwordHash = hash
}

func (u *User) SetRole(role Role) {
	u.role = role
}

func (u *User) SetAvatarURL(url string) {
	u.avatarURL = url
}

func (u *User) SetName(name string) {
	u.name = name
}

// SetGoogleSub links an existing account (created via magic link, or an
// invited admin) to a Google identity the first time they sign in with
// Google using the same email — see application.resolveOAuthUser.
func (u *User) SetGoogleSub(sub string) {
	u.googleSub = &sub
}

// UpdateProfile applies a self-service edit of display name and email.
// Unlike CanManageUser/UpdateUser (an admin acting on someone else, gated by
// role rank), there is no privilege check here — every account, including a
// plain "user", is always allowed to edit its own name/email. Email
// uniqueness against *other* accounts can't be checked here (needs a repo
// call) — that's the application layer's job before calling this.
func (u *User) UpdateProfile(name, email string) error {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	if errs := validateEmail(email); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"email": errs})
	}

	u.name = name
	u.email = email
	return nil
}

func (u *User) RecordLogin(at time.Time) {
	u.lastLoginAt = &at
}

func (u *User) Disable() {
	u.status = UserStatusDisabled
}

func (u *User) Activate() {
	u.status = UserStatusActive
}

func validateUsername(username string) []string {
	var errs []string
	if len(username) < 3 || len(username) > 32 {
		errs = append(errs, "username must be between 3 and 32 characters")
	}
	if username != "" && !usernamePattern.MatchString(username) {
		errs = append(errs, "username may only contain letters, numbers, dots and underscores")
	}
	return errs
}

func validateEmail(email string) []string {
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return []string{"invalid email"}
	}
	return nil
}
