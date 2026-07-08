package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/review/domain"
)

func TestCreateReview(t *testing.T) {
	repo := newFakeReviewRepository()
	ctx := context.Background()

	productID := "product-1"
	input := domain.CreateReviewInput{
		ProductID:    &productID,
		Rating:       5,
		Comment:      "Máy chạy êm, lắp đặt nhanh, rất hài lòng.",
		ReviewerName: "Nguyen Van A",
	}

	review, err := CreateReview(ctx, repo, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if review.ID() == "" {
		t.Error("expected review to have an ID after create")
	}
	if !review.IsPublished() {
		t.Error("expected a clean review to be published by default")
	}
}

func TestCreateReview_ValidationError(t *testing.T) {
	repo := newFakeReviewRepository()
	ctx := context.Background()

	_, err := CreateReview(ctx, repo, domain.CreateReviewInput{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestCreateReview_RejectsBothEntityRefs(t *testing.T) {
	repo := newFakeReviewRepository()
	ctx := context.Background()

	productID := "product-1"
	serviceID := "service-1"
	_, err := CreateReview(ctx, repo, domain.CreateReviewInput{
		ProductID:    &productID,
		ServiceID:    &serviceID,
		Rating:       5,
		Comment:      "Good",
		ReviewerName: "A",
	})
	if err == nil {
		t.Fatal("expected validation error when both productId and serviceId are set")
	}
}

func TestCreateReview_AutoHidesBlockedContent(t *testing.T) {
	repo := newFakeReviewRepository()
	ctx := context.Background()

	productID := "product-1"
	input := domain.CreateReviewInput{
		ProductID:    &productID,
		Rating:       1,
		Comment:      "Liên hệ zalo: 0901234567 để vay tiền nhanh",
		ReviewerName: "Spammer",
	}

	review, err := CreateReview(ctx, repo, input)
	if err != nil {
		t.Fatalf("expected no error (auto-hide, not reject), got %v", err)
	}
	if review.IsPublished() {
		t.Error("expected spam-flagged review to be auto-hidden (IsPublished=false)")
	}
}
