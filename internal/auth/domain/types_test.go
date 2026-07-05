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

func TestUser_CanInvite(t *testing.T) {
	user, _ := NewUser("user1", "user1@example.com", "hash", "", "", RoleUser)
	admin, _ := NewUser("admin1", "admin1@example.com", "hash", "", "", RoleAdmin)
	superAdmin, _ := NewUser("root", "root@example.com", "hash", "", "", RoleSuperAdmin)

	if user.CanInvite(RoleUser) {
		t.Error("plain user role must not be able to invite anyone")
	}
	if !admin.CanInvite(RoleAdmin) {
		t.Error("admin should be able to invite another admin")
	}
	if !admin.CanInvite(RoleUser) {
		t.Error("admin should be able to invite a user")
	}
	if admin.CanInvite(RoleSuperAdmin) {
		t.Error("admin must not be able to invite a super_admin")
	}
	if !superAdmin.CanInvite(RoleSuperAdmin) {
		t.Error("super_admin should be able to invite another super_admin")
	}
	if !superAdmin.CanInvite(RoleAdmin) {
		t.Error("super_admin should be able to invite an admin")
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

func TestValidatePassword(t *testing.T) {
	valid := []string{"Vlu15112002@", "Str0ng!Passw0rd"}
	for _, p := range valid {
		if errs := ValidatePassword(p); len(errs) > 0 {
			t.Errorf("expected %q to be valid, got errors: %v", p, errs)
		}
	}

	invalid := map[string]string{
		"short1A!":       "too short (8 chars, needs more than 8)",
		"alllowercase1!": "missing uppercase",
		"ALLUPPERCASE1!": "missing lowercase",
		"NoDigitsHere!":  "missing digit",
		"NoSpecial1234":  "missing special character",
	}
	for p, reason := range invalid {
		if errs := ValidatePassword(p); len(errs) == 0 {
			t.Errorf("expected %q to be invalid (%s)", p, reason)
		}
	}
}
