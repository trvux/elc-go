package domain

// Permission is a fine-grained capability, checked independently of Role at
// the call site — handlers/use cases ask "can this role do X", not "is this
// role admin", so the answer can change by editing rolePermissions in one
// place instead of hunting down every `role == "admin"` comparison.
//
// This is intentionally a static Go map, not a DB-backed policy table: the
// role set is small and fixed (super_admin/admin/user), nobody has asked for
// admins to redefine permissions at runtime, and every other piece of this
// codebase avoids infrastructure a real requirement hasn't shown up for yet.
// If per-account custom permissions are ever needed, replace rolePermissions
// with a `role_permissions` table behind the same HasPermission signature —
// nothing calling HasPermission needs to change.
type Permission string

const (
	// PermissionUsersManage covers inviting, disabling, and changing the
	// role of other admin-panel accounts.
	PermissionUsersManage Permission = "users:manage"
	// PermissionContentWrite covers day-to-day content modules (catalog,
	// news, pages, branches, ...) once they adopt RequirePermission.
	PermissionContentWrite Permission = "content:write"
	// PermissionContentDelete is split out from PermissionContentWrite so a
	// role can create/edit content without being able to delete it.
	PermissionContentDelete Permission = "content:delete"
	// PermissionSettingsManage covers site-wide settings, not per-record
	// content.
	PermissionSettingsManage Permission = "settings:manage"
	// PermissionContentPublish covers moving content through an
	// approval gate (e.g. product proposed -> published/rejected,
	// published -> archived) — split from PermissionContentWrite so a
	// role can create/edit drafts without being able to publish them.
	PermissionContentPublish Permission = "content:publish"
)

// rolePermissions lists what each non-super_admin role grants. super_admin
// is deliberately not listed here — see HasPermission.
var rolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermissionUsersManage,
		PermissionContentWrite,
		PermissionContentDelete,
		PermissionContentPublish,
		PermissionSettingsManage,
	},
	RoleUser: {
		PermissionContentWrite,
	},
}

// HasPermission reports whether the role grants perm. super_admin always
// returns true unconditionally, rather than being enumerated in
// rolePermissions — an entry there would just have to be remembered and kept
// in sync by hand every time a new permission constant is added.
func (r Role) HasPermission(perm Permission) bool {
	if r == RoleSuperAdmin {
		return true
	}
	for _, p := range rolePermissions[r] {
		if p == perm {
			return true
		}
	}
	return false
}

// CanWriteContent and CanDeleteContent are ready-made predicates for other
// modules' presentation/routes.go — every business module's create/update
// routes want the exact same "content:write" check, and delete routes want
// "content:delete", so each module doesn't need to write its own closure.
// Signature matches httpserver.RequirePermission's `func(role string) bool`.
func CanWriteContent(role string) bool {
	return Role(role).HasPermission(PermissionContentWrite)
}

func CanDeleteContent(role string) bool {
	return Role(role).HasPermission(PermissionContentDelete)
}

// CanPublishContent gates approval-workflow transitions (approve/reject/
// archive) that move content out of the draft an ordinary editor can reach.
func CanPublishContent(role string) bool {
	return Role(role).HasPermission(PermissionContentPublish)
}

// CanManageSettings is for the settings module (and any future
// site-wide-config route) — distinct from per-record content permissions.
func CanManageSettings(role string) bool {
	return Role(role).HasPermission(PermissionSettingsManage)
}

// CanManageAccounts gates the invite-creation and user-management routes.
func CanManageAccounts(role string) bool {
	return Role(role).HasPermission(PermissionUsersManage)
}
