package domain

// Role gates what an account can do. There is no "customer"/public role yet —
// this module only serves the admin panel; a public-facing role can be added
// later without touching admin semantics.
type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	// RoleUser is the lowest admin-panel role — day-to-day content work
	// (catalog, news, pages, ...) without account management or settings
	// access. Not a public/customer role.
	RoleUser Role = "user"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleUser:
		return true
	default:
		return false
	}
}

// roleRank orders roles by privilege, used by User.CanInvite to enforce that
// nobody can grant a role higher than their own.
var roleRank = map[Role]int{
	RoleUser:       1,
	RoleAdmin:      2,
	RoleSuperAdmin: 3,
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)
