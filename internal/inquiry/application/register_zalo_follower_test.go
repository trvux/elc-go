package application

import (
	"context"
	"testing"
)

func TestRegisterZaloFollower(t *testing.T) {
	repo := newFakeZaloFollowerRepository()
	ctx := context.Background()

	name := "Bảo Huy"
	if err := RegisterZaloFollower(ctx, repo, "zalo-user-1", &name); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	active, err := repo.GetAllActive(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(active) != 1 || active[0].ZaloUserID() != "zalo-user-1" {
		t.Errorf("expected 1 active follower zalo-user-1, got %+v", active)
	}
}

func TestDeactivateZaloFollower(t *testing.T) {
	repo := newFakeZaloFollowerRepository()
	ctx := context.Background()

	_ = RegisterZaloFollower(ctx, repo, "zalo-user-1", nil)
	if err := DeactivateZaloFollower(ctx, repo, "zalo-user-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	active, err := repo.GetAllActive(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(active) != 0 {
		t.Errorf("expected 0 active followers after deactivate, got %+v", active)
	}
}
