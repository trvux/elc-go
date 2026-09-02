package domain

// Role gates what an account can do. RoleMember is the public role: anyone
// who signs in via Google or magic link gets it automatically, with zero
// admin-panel permissions (see permission.go). RoleUser/RoleAdmin/
// RoleSuperAdmin are the admin-panel tiers — never assigned automatically,
// always a deliberate promotion of an existing (already-logged-in) account by
// someone who outranks the target role.
type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	// RoleUser is the lowest admin-panel role — day-to-day content work
	// (catalog, news, pages, ...) without account management or settings
	// access. Not a public/customer role.
	RoleUser Role = "user"
	// RoleMember is the public role — no admin-panel permissions at all.
	RoleMember Role = "member"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleUser, RoleMember:
		return true
	default:
		return false
	}
}

// roleRank orders roles by privilege, used by User.CanGrantRole to enforce
// that nobody can grant a role higher than their own.
var roleRank = map[Role]int{
	RoleMember:     0,
	RoleUser:       1,
	RoleAdmin:      2,
	RoleSuperAdmin: 3,
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)
