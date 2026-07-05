package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func seedActiveUser(t *testing.T, repo *fakeUserRepository, username, email, password string, role domain.Role) *domain.User {
	t.Helper()
	hasher := fakePasswordHasher{}
	hash, _ := hasher.Hash(password)
	user, err := domain.NewUser(username, email, hash, "", "", role)
	if err != nil {
		t.Fatalf("failed to build user: %v", err)
	}
	return mustCreate(t, repo, user)
}

func TestLogin_ByUsername(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}
	issuer := fakeTokenIssuer{}
	ctx := context.Background()

	seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "Vlu15112002@", domain.RoleSuperAdmin)

	result, err := Login(ctx, userRepo, sessionRepo, hasher, issuer, LoginInput{
		Identifier: "tranvux",
		Password:   "Vlu15112002@",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Error("expected both access and refresh tokens to be issued")
	}
}

func TestLogin_ByEmail(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}
	issuer := fakeTokenIssuer{}
	ctx := context.Background()

	seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "Vlu15112002@", domain.RoleSuperAdmin)

	result, err := Login(ctx, userRepo, sessionRepo, hasher, issuer, LoginInput{
		Identifier: "tranvux@example.com",
		Password:   "Vlu15112002@",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.User.Username() != "tranvux" {
		t.Errorf("expected to resolve to tranvux, got %s", result.User.Username())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}
	issuer := fakeTokenIssuer{}
	ctx := context.Background()

	seedActiveUser(t, userRepo, "tranvux", "tranvux@example.com", "Vlu15112002@", domain.RoleSuperAdmin)

	if _, err := Login(ctx, userRepo, sessionRepo, hasher, issuer, LoginInput{
		Identifier: "tranvux",
		Password:   "WrongPassword1!",
	}); err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLogin_UnknownIdentifier(t *testing.T) {
	userRepo := newFakeUserRepository()
	sessionRepo := newFakeSessionRepository()
	hasher := fakePasswordHasher{}
	issuer := fakeTokenIssuer{}
	ctx := context.Background()

	if _, err := Login(ctx, userRepo, sessionRepo, hasher, issuer, LoginInput{
		Identifier: "ghost",
		Password:   "Vlu15112002@",
	}); err == nil {
		t.Fatal("expected error for unknown identifier")
	}
}
