package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/faq/domain"
)

func TestCreateFAQ_CreatesPublishedByDefault(t *testing.T) {
	repo := newFakeFAQRepository()
	ctx := context.Background()

	faq, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType:   domain.OwnerTypeService,
		OwnerID:     "service-1",
		Question:    "Máy lạnh Daikin báo lỗi U4 là gì?",
		Answer:      "U4 là lỗi đường truyền giữa dàn lạnh và dàn nóng.",
		OrderIndex:  0,
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if faq.OwnerType() != domain.OwnerTypeService || faq.OwnerID() != "service-1" {
		t.Errorf("expected owner service-1, got %s/%s", faq.OwnerType(), faq.OwnerID())
	}
	if !faq.IsPublished() {
		t.Error("expected published FAQ")
	}
}

func TestCreateFAQ_RejectsInvalidOwnerType(t *testing.T) {
	repo := newFakeFAQRepository()
	ctx := context.Background()

	_, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType: "garbage",
		OwnerID:   "1",
		Question:  "q",
		Answer:    "a",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid owner type")
	}
}

func TestCreateFAQ_RejectsBlankQuestionOrAnswer(t *testing.T) {
	repo := newFakeFAQRepository()
	ctx := context.Background()

	if _, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType: domain.OwnerTypeService, OwnerID: "1", Question: "", Answer: "a",
	}); err == nil {
		t.Error("expected validation error for blank question")
	}
	if _, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType: domain.OwnerTypeService, OwnerID: "1", Question: "q", Answer: "",
	}); err == nil {
		t.Error("expected validation error for blank answer")
	}
}

func TestGetFAQsByOwner_PublishedOnlyExcludesDrafts(t *testing.T) {
	repo := newFakeFAQRepository()
	ctx := context.Background()

	if _, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType: domain.OwnerTypeService, OwnerID: "service-1",
		Question: "q1", Answer: "a1", OrderIndex: 0, IsPublished: true,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType: domain.OwnerTypeService, OwnerID: "service-1",
		Question: "q2 (draft)", Answer: "a2", OrderIndex: 1, IsPublished: false,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	published, err := GetFAQsByOwner(ctx, repo, domain.FAQFilter{
		OwnerType: domain.OwnerTypeService, OwnerID: "service-1", PublishedOnly: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(published) != 1 || published[0].Question() != "q1" {
		t.Errorf("expected only the published FAQ, got %d: %+v", len(published), published)
	}

	all, err := GetFAQsByOwner(ctx, repo, domain.FAQFilter{
		OwnerType: domain.OwnerTypeService, OwnerID: "service-1", PublishedOnly: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected both FAQs for admin view, got %d", len(all))
	}
}

func TestGetFAQsByOwner_DifferentOwnersDoNotLeak(t *testing.T) {
	repo := newFakeFAQRepository()
	ctx := context.Background()

	if _, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType: domain.OwnerTypeService, OwnerID: "service-1",
		Question: "q", Answer: "a", IsPublished: true,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType: domain.OwnerTypeProduct, OwnerID: "service-1",
		Question: "q", Answer: "a", IsPublished: true,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := GetFAQsByOwner(ctx, repo, domain.FAQFilter{
		OwnerType: domain.OwnerTypeService, OwnerID: "service-1", PublishedOnly: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected same owner_id but different owner_type not to leak across, got %d", len(result))
	}
}

func TestUpdateFAQ_UpdatesOnlyGivenFields(t *testing.T) {
	repo := newFakeFAQRepository()
	ctx := context.Background()

	faq, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType: domain.OwnerTypeService, OwnerID: "service-1",
		Question: "q", Answer: "a", OrderIndex: 0, IsPublished: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newAnswer := "a2"
	published := true
	updated, err := UpdateFAQ(ctx, repo, domain.UpdateFAQInput{
		ID: faq.ID(), Answer: &newAnswer, IsPublished: &published,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Question() != "q" {
		t.Errorf("expected question unchanged, got %q", updated.Question())
	}
	if updated.Answer() != "a2" {
		t.Errorf("expected answer updated, got %q", updated.Answer())
	}
	if !updated.IsPublished() {
		t.Error("expected published=true after update")
	}
}

func TestUpdateFAQ_NotFound(t *testing.T) {
	repo := newFakeFAQRepository()
	ctx := context.Background()

	_, err := UpdateFAQ(ctx, repo, domain.UpdateFAQInput{ID: "does-not-exist"})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteFAQ_RemovesRow(t *testing.T) {
	repo := newFakeFAQRepository()
	ctx := context.Background()

	faq, err := CreateFAQ(ctx, repo, domain.CreateFAQInput{
		OwnerType: domain.OwnerTypeService, OwnerID: "service-1",
		Question: "q", Answer: "a", IsPublished: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := DeleteFAQ(ctx, repo, faq.ID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := GetFAQByID(ctx, repo, faq.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected FAQ to be gone after delete")
	}
}
