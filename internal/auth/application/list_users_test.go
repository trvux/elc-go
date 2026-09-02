package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/auth/domain"
)

func TestListUsers(t *testing.T) {
	userRepo := newFakeUserRepository()
	ctx := context.Background()

	seedActiveUser(t, userRepo, "root", "root@example.com", domain.RoleSuperAdmin)
	seedActiveUser(t, userRepo, "admin1", "admin1@example.com", domain.RoleAdmin)

	users, err := ListUsers(ctx, userRepo)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}
