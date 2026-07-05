package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestUpdateUser_ChangeRole(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	ctx := context.Background()

	superAdmin := seedActiveUser(t, userRepo, "root", "root@example.com", "Vlu15112002@", domain.RoleSuperAdmin)
	target := seedActiveUser(t, userRepo, "user1", "user1@example.com", "Vlu15112002@", domain.RoleUser)

	newRole := domain.RoleAdmin
	updated, err := UpdateUser(ctx, userRepo, sessionRepo, superAdmin, target.ID(), UpdateUserInput{Role: &newRole})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Role() != domain.RoleAdmin {
		t.Errorf("expected role to be updated to admin, got %s", updated.Role())
	}
}

func TestUpdateUser_CannotEscalateAboveOwnRank(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	ctx := context.Background()

	admin := seedActiveUser(t, userRepo, "admin1", "admin1@example.com", "Vlu15112002@", domain.RoleAdmin)
	target := seedActiveUser(t, userRepo, "user1", "user1@example.com", "Vlu15112002@", domain.RoleUser)

	newRole := domain.RoleSuperAdmin
	if _, err := UpdateUser(ctx, userRepo, sessionRepo, admin, target.ID(), UpdateUserInput{Role: &newRole}); err == nil {
		t.Fatal("expected error when admin tries to promote someone to super_admin")
	}
}

func TestUpdateUser_CannotManageHigherRankedUser(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	ctx := context.Background()

	admin := seedActiveUser(t, userRepo, "admin1", "admin1@example.com", "Vlu15112002@", domain.RoleAdmin)
	superAdmin := seedActiveUser(t, userRepo, "root", "root@example.com", "Vlu15112002@", domain.RoleSuperAdmin)

	status := domain.UserStatusDisabled
	if _, err := UpdateUser(ctx, userRepo, sessionRepo, admin, superAdmin.ID(), UpdateUserInput{Status: &status}); err == nil {
		t.Fatal("expected error when admin tries to disable a super_admin")
	}
}

func TestUpdateUser_CannotActOnSelf(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	ctx := context.Background()

	superAdmin := seedActiveUser(t, userRepo, "root", "root@example.com", "Vlu15112002@", domain.RoleSuperAdmin)

	status := domain.UserStatusDisabled
	if _, err := UpdateUser(ctx, userRepo, sessionRepo, superAdmin, superAdmin.ID(), UpdateUserInput{Status: &status}); err == nil {
		t.Fatal("expected error when acting on your own account")
	}
}

func TestUpdateUser_DisableRevokesAllSessions(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	ctx := context.Background()

	superAdmin := seedActiveUser(t, userRepo, "root", "root@example.com", "Vlu15112002@", domain.RoleSuperAdmin)
	target := seedActiveUser(t, userRepo, "user1", "user1@example.com", "Vlu15112002@", domain.RoleUser)

	session, _, err := domain.NewSession(target.ID(), "", "", RefreshTokenTTL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	created, err := sessionRepo.Create(ctx, session)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	status := domain.UserStatusDisabled
	if _, err := UpdateUser(ctx, userRepo, sessionRepo, superAdmin, target.ID(), UpdateUserInput{Status: &status}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sessionRepo.sessions[created.ID()].IsValid() {
		t.Error("expected the disabled user's session to be revoked")
	}
}

func TestUpdateUser_Reactivate(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	ctx := context.Background()

	superAdmin := seedActiveUser(t, userRepo, "root", "root@example.com", "Vlu15112002@", domain.RoleSuperAdmin)
	target := seedActiveUser(t, userRepo, "user1", "user1@example.com", "Vlu15112002@", domain.RoleUser)
	target.Disable()
	if _, err := userRepo.Update(ctx, target); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	status := domain.UserStatusActive
	updated, err := UpdateUser(ctx, userRepo, sessionRepo, superAdmin, target.ID(), UpdateUserInput{Status: &status})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !updated.IsActive() {
		t.Error("expected user to be active again")
	}
}
