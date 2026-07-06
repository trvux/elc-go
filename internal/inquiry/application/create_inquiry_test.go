package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

func TestCreateInquiry(t *testing.T) {
	repo := newFakeInquiryRepository()
	notifier := &fakeLeadNotifier{}
	ctx := context.Background()

	input := domain.CreateInquiryInput{
		Name:  "Nguyen Van A",
		Phone: "0901234567",
	}

	inquiry, err := CreateInquiry(ctx, repo, notifier, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inquiry.ID() == "" {
		t.Error("expected inquiry to have an ID after create")
	}
	if inquiry.Status() != domain.InquiryStatusNew {
		t.Errorf("expected status new, got %s", inquiry.Status())
	}
	if len(notifier.notified) != 1 {
		t.Errorf("expected notifier to be called once, got %d", len(notifier.notified))
	}
}

func TestCreateInquiry_ValidationError(t *testing.T) {
	repo := newFakeInquiryRepository()
	notifier := &fakeLeadNotifier{}
	ctx := context.Background()

	_, err := CreateInquiry(ctx, repo, notifier, domain.CreateInquiryInput{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if len(notifier.notified) != 0 {
		t.Error("expected notifier not to be called when create fails")
	}
}

func TestCreateInquiry_RejectsMultipleEntityRefs(t *testing.T) {
	repo := newFakeInquiryRepository()
	notifier := &fakeLeadNotifier{}
	ctx := context.Background()

	productID := "product-1"
	projectID := "project-1"
	_, err := CreateInquiry(ctx, repo, notifier, domain.CreateInquiryInput{
		Name:      "Nguyen Van A",
		Phone:     "0901234567",
		ProductID: &productID,
		ProjectID: &projectID,
	})
	if err == nil {
		t.Fatal("expected validation error when both productId and projectId are set")
	}
}
