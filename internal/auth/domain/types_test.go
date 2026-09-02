package domain

import "testing"

func TestNewUser(t *testing.T) {
	user, err := NewUser("tranvux", "TranVu@Example.com", "bcrypt-hash", "Bảo Huy", "0909411633", RoleSuperAdmin)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Username() != "tranvux" {
		t.Errorf("expected lowercase username, got %s", user.Username())
	}
	if user.Email() != "tranvu@example.com" {
		t.Errorf("expected lowercase email, got %s", user.Email())
	}
	if user.Name() != "Bảo Huy" {
		t.Errorf("expected name to be preserved, got %s", user.Name())
	}
	if !user.IsActive() {
		t.Error("expected new user to be active")
	}
}

func TestNewUser_NameAndPhoneAreOptional(t *testing.T) {
	user, err := NewUser("tranvux", "a@b.com", "hash", "", "", RoleAdmin)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Name() != "" || user.Phone() != "" {
		t.Errorf("expected empty name/phone to be accepted, got name=%q phone=%q", user.Name(), user.Phone())
	}
}

func TestNewUser_ValidationError(t *testing.T) {
	cases := []struct {
		name         string
		username     string
		email        string
		passwordHash string
		role         Role
	}{
		{"short username", "ab", "a@b.com", "hash", RoleAdmin},
		{"bad username chars", "trần vũ", "a@b.com", "hash", RoleAdmin},
		{"invalid email", "tranvux", "not-an-email", "hash", RoleAdmin},
		{"missing password hash", "tranvux", "a@b.com", "", RoleAdmin},
		{"invalid role", "tranvux", "a@b.com", "hash", Role("owner")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewUser(tc.username, tc.email, tc.passwordHash, "", "", tc.role); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestUser_CanGrantRole(t *testing.T) {
	member, _ := NewUser("member1", "member1@example.com", "hash", "", "", RoleMember)
	user, _ := NewUser("user1", "user1@example.com", "hash", "", "", RoleUser)
	admin, _ := NewUser("admin1", "admin1@example.com", "hash", "", "", RoleAdmin)
	superAdmin, _ := NewUser("root", "root@example.com", "hash", "", "", RoleSuperAdmin)

	if member.CanGrantRole(RoleMember) {
		t.Error("member role must not be able to grant anyone a role")
	}
	if user.CanGrantRole(RoleUser) {
		t.Error("plain user role must not be able to grant anyone a role")
	}
	if !admin.CanGrantRole(RoleAdmin) {
		t.Error("admin should be able to grant another admin")
	}
	if !admin.CanGrantRole(RoleUser) {
		t.Error("admin should be able to grant a user")
	}
	if admin.CanGrantRole(RoleSuperAdmin) {
		t.Error("admin must not be able to grant a super_admin")
	}
	if !superAdmin.CanGrantRole(RoleSuperAdmin) {
		t.Error("super_admin should be able to grant another super_admin")
	}
	if !superAdmin.CanGrantRole(RoleAdmin) {
		t.Error("super_admin should be able to grant an admin")
	}
}

func TestUser_CanManageUser(t *testing.T) {
	user, _ := NewUser("user1", "user1@example.com", "hash", "", "", RoleUser)
	admin, _ := NewUser("admin1", "admin1@example.com", "hash", "", "", RoleAdmin)
	superAdmin, _ := NewUser("root", "root@example.com", "hash", "", "", RoleSuperAdmin)

	if !admin.CanManageUser(user) {
		t.Error("admin should be able to manage a plain user")
	}
	if !admin.CanManageUser(admin) {
		t.Error("admin should be able to manage a same-rank admin")
	}
	if admin.CanManageUser(superAdmin) {
		t.Error("admin must not be able to manage a super_admin")
	}
	if !superAdmin.CanManageUser(admin) {
		t.Error("super_admin should be able to manage an admin")
	}
	if admin.CanManageUser(nil) {
		t.Error("managing a nil target must never be allowed")
	}
}

func TestUser_ActivateDisable(t *testing.T) {
	user, _ := NewUser("user1", "user1@example.com", "hash", "", "", RoleUser)
	if !user.IsActive() {
		t.Fatal("expected new user to start active")
	}
	user.Disable()
	if user.IsActive() {
		t.Error("expected user to be inactive after Disable")
	}
	user.Activate()
	if !user.IsActive() {
		t.Error("expected user to be active again after Activate")
	}
}

func TestUser_SetRole(t *testing.T) {
	user, _ := NewUser("user1", "user1@example.com", "hash", "", "", RoleUser)
	user.SetRole(RoleAdmin)
	if user.Role() != RoleAdmin {
		t.Errorf("expected role to be updated to admin, got %s", user.Role())
	}
}

func TestRole_HasPermission(t *testing.T) {
	if !RoleSuperAdmin.HasPermission(PermissionUsersManage) {
		t.Error("super_admin should have every permission")
	}
	if !RoleAdmin.HasPermission(PermissionUsersManage) {
		t.Error("admin should be able to manage users")
	}
	if RoleUser.HasPermission(PermissionUsersManage) {
		t.Error("plain user role must not be able to manage users")
	}
	if !RoleUser.HasPermission(PermissionContentWrite) {
		t.Error("plain user role should still be able to write content")
	}
	if RoleUser.HasPermission(PermissionContentDelete) {
		t.Error("plain user role must not be able to delete content")
	}
}

func TestNewOAuthUser(t *testing.T) {
	sub := "google-sub-123"
	user, err := NewOAuthUser("New.Member@Example.com", "Bảo Huy", "https://example.com/avatar.png", &sub, RoleMember)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email() != "new.member@example.com" {
		t.Errorf("expected lowercase email, got %s", user.Email())
	}
	if user.Username() != "" {
		t.Errorf("expected no username for an OAuth user, got %q", user.Username())
	}
	if user.PasswordHash() != "" {
		t.Errorf("expected no password hash for an OAuth user, got %q", user.PasswordHash())
	}
	if user.GoogleSub() == nil || *user.GoogleSub() != sub {
		t.Errorf("expected google sub to be preserved, got %v", user.GoogleSub())
	}
	if user.Role() != RoleMember {
		t.Errorf("expected role to be preserved, got %s", user.Role())
	}
	if !user.IsActive() {
		t.Error("expected new OAuth user to be active")
	}
}

func TestNewOAuthUser_MagicLinkOnlyHasNoGoogleSub(t *testing.T) {
	user, err := NewOAuthUser("member@example.com", "", "", nil, RoleMember)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.GoogleSub() != nil {
		t.Errorf("expected no google sub for a magic-link-only user, got %v", user.GoogleSub())
	}
}

func TestNewOAuthUser_ValidationError(t *testing.T) {
	if _, err := NewOAuthUser("not-an-email", "", "", nil, RoleMember); err == nil {
		t.Error("expected validation error for bad email")
	}
	if _, err := NewOAuthUser("a@b.com", "", "", nil, Role("owner")); err == nil {
		t.Error("expected validation error for bad role")
	}
}
