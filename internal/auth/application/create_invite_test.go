package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestCreateInvite(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	emailSender := &fakeEmailSender{}
	ctx := context.Background()

	superAdmin, _ := domain.NewUser("root", "root@example.com", "hash", "", "", domain.RoleSuperAdmin)
	superAdmin = mustCreate(t, userRepo, superAdmin)

	token, err := CreateInvite(ctx, userRepo, tokenRepo, emailSender, superAdmin, CreateInviteInput{
		Email: "new.admin@example.com",
		Role:  domain.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token.Email() != "new.admin@example.com" {
		t.Errorf("unexpected token email: %s", token.Email())
	}
	if len(emailSender.invitesSent) != 1 || emailSender.invitesSent[0] != "new.admin@example.com" {
		t.Errorf("expected invite email to be sent, got %v", emailSender.invitesSent)
	}
}

func TestCreateInvite_PrivilegeEscalationBlocked(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	emailSender := &fakeEmailSender{}
	ctx := context.Background()

	admin, _ := domain.NewUser("admin1", "admin1@example.com", "hash", "", "", domain.RoleAdmin)
	admin = mustCreate(t, userRepo, admin)

	_, err := CreateInvite(ctx, userRepo, tokenRepo, emailSender, admin, CreateInviteInput{
		Email: "wannabe-root@example.com",
		Role:  domain.RoleSuperAdmin,
	})
	if err == nil {
		t.Fatal("expected an admin inviting a super_admin to be rejected")
	}
}

func TestCreateInvite_EmailAlreadyRegistered(t *testing.T) {
	userRepo := newFakeUserRepository()
	tokenRepo := newFakeTokenRepository()
	emailSender := &fakeEmailSender{}
	ctx := context.Background()

	superAdmin, _ := domain.NewUser("root", "root@example.com", "hash", "", "", domain.RoleSuperAdmin)
	superAdmin = mustCreate(t, userRepo, superAdmin)
	existing, _ := domain.NewUser("existing", "existing@example.com", "hash", "", "", domain.RoleAdmin)
	mustCreate(t, userRepo, existing)

	_, err := CreateInvite(ctx, userRepo, tokenRepo, emailSender, superAdmin, CreateInviteInput{
		Email: "existing@example.com",
		Role:  domain.RoleAdmin,
	})
	if err == nil {
		t.Fatal("expected error when inviting an already-registered email")
	}
}

func mustCreate(t *testing.T, repo *fakeUserRepository, user *domain.User) *domain.User {
	t.Helper()
	created, err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	return created
}
