package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/event/domain"
)

func TestLogEvent(t *testing.T) {
	repo := newFakeEventRepository()
	ctx := context.Background()

	entityType := domain.EntityTypeProduct
	entityID := "product-1"
	pagePath := "/san-pham/product-1"
	sessionID := "session-1"

	err := LogEvent(ctx, repo, domain.CreateEventInput{
		Name:       domain.EventViewItem,
		EntityType: &entityType,
		EntityID:   &entityID,
		PagePath:   &pagePath,
		SessionID:  &sessionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.events) != 1 {
		t.Errorf("expected 1 event stored, got %d", len(repo.events))
	}
}

func TestLogEvent_InvalidName(t *testing.T) {
	repo := newFakeEventRepository()
	ctx := context.Background()

	err := LogEvent(ctx, repo, domain.CreateEventInput{Name: "bogus"})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestGetTopViewed(t *testing.T) {
	repo := newFakeEventRepository()
	ctx := context.Background()

	entityType := domain.EntityTypeProduct
	productA := "product-a"
	productB := "product-b"

	for i := 0; i < 3; i++ {
		_ = LogEvent(ctx, repo, domain.CreateEventInput{Name: domain.EventViewItem, EntityType: &entityType, EntityID: &productA})
	}
	_ = LogEvent(ctx, repo, domain.CreateEventInput{Name: domain.EventViewItem, EntityType: &entityType, EntityID: &productB})

	result, err := GetTopViewed(ctx, repo, domain.TopViewedFilter{EntityType: domain.EntityTypeProduct})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 distinct entities, got %d", len(result))
	}
}
