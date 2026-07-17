package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

func TestCreateInquiry(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	input := domain.CreateInquiryInput{
		Name:  "Nguyen Van A",
		Phone: "0901234567",
	}

	inquiry, err := CreateInquiry(ctx, repo, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inquiry.ID() == "" {
		t.Error("expected inquiry to have an ID after create")
	}
	if inquiry.Status() != domain.InquiryStatusNew {
		t.Errorf("expected status new, got %s", inquiry.Status())
	}
}

func TestCreateInquiry_ValidationError(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	_, err := CreateInquiry(ctx, repo, domain.CreateInquiryInput{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestCreateInquiry_RejectsMultipleEntityRefs(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	productID := "product-1"
	projectID := "project-1"
	_, err := CreateInquiry(ctx, repo, domain.CreateInquiryInput{
		Name:      "Nguyen Van A",
		Phone:     "0901234567",
		ProductID: &productID,
		ProjectID: &projectID,
	})
	if err == nil {
		t.Fatal("expected validation error when both productId and projectId are set")
	}
}
