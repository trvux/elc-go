package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._]+$`)
	upperPattern    = regexp.MustCompile(`[A-Z]`)
	lowerPattern    = regexp.MustCompile(`[a-z]`)
	digitPattern    = regexp.MustCompile(`[0-9]`)
	specialPattern  = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

type User struct {
	id           string
	username     string
	email        string
	passwordHash string
	name         string
	phone        string
	avatarURL    string
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

// RehydrateUser reconstructs a User from a trusted DB row. No validation —
// only the infrastructure layer should call this.
func RehydrateUser(
	id, username, email, passwordHash, name, phone, avatarURL string,
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
func (u *User) Role() Role              { return u.role }
func (u *User) Status() UserStatus      { return u.status }
func (u *User) LastLoginAt() *time.Time { return u.lastLoginAt }
func (u *User) CreatedAt() time.Time    { return u.createdAt }
func (u *User) UpdatedAt() time.Time    { return u.updatedAt }
func (u *User) IsActive() bool          { return u.status == UserStatusActive }

// CanInvite enforces the privilege-escalation guard: nobody can grant a role
// ranked higher than their own (see roleRank), and the lowest role (user)
// cannot invite anyone at all — inviting is account management, not content
// work.
func (u *User) CanInvite(targetRole Role) bool {
	if !targetRole.IsValid() || u.role == RoleUser {
		return false
	}
	return roleRank[u.role] >= roleRank[targetRole]
}

// CanManageUser enforces the same privilege boundary as CanInvite but for an
// *existing* account: nobody can change the role/status of a user who
// currently outranks them. Combine with CanInvite(newRole) when the action
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

// ValidatePassword enforces the password policy shared by accept-invite and
// reset-password: more than 8 characters, at least one uppercase, one
// lowercase, one digit, and one special character.
func ValidatePassword(password string) []string {
	var errs []string
	if len(password) <= 8 {
		errs = append(errs, "password must be more than 8 characters")
	}
	if !upperPattern.MatchString(password) {
		errs = append(errs, "password must contain at least one uppercase letter")
	}
	if !lowerPattern.MatchString(password) {
		errs = append(errs, "password must contain at least one lowercase letter")
	}
	if !digitPattern.MatchString(password) {
		errs = append(errs, "password must contain at least one digit")
	}
	if !specialPattern.MatchString(password) {
		errs = append(errs, "password must contain at least one special character")
	}
	return errs
}
