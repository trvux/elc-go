package domain

import (
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

func TestNewReview(t *testing.T) {
	productID := "product-1"

	t.Run("valid input creates a published review", func(t *testing.T) {
		r, err := NewReview(&productID, nil, 5, "Rất tốt, đóng gói cẩn thận.", "Nguyen Van A", nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Rating() != 5 {
			t.Errorf("expected rating 5, got %d", r.Rating())
		}
		if !r.IsPublished() {
			t.Error("expected clean review to be published by default")
		}
	})

	t.Run("neither productId nor serviceId fails validation", func(t *testing.T) {
		_, err := NewReview(nil, nil, 5, "Good", "A", nil, nil, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
		if _, ok := appErr.Fields["entity"]; !ok {
			t.Errorf("expected validation field for entity, got: %+v", appErr.Fields)
		}
	})

	t.Run("both productId and serviceId fails validation", func(t *testing.T) {
		serviceID := "service-1"
		_, err := NewReview(&productID, &serviceID, 5, "Good", "A", nil, nil, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("rating out of range fails validation", func(t *testing.T) {
		_, err := NewReview(&productID, nil, 6, "Good", "A", nil, nil, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
		if _, ok := appErr.Fields["rating"]; !ok {
			t.Errorf("expected validation field for rating, got: %+v", appErr.Fields)
		}
	})

	t.Run("empty comment fails validation", func(t *testing.T) {
		_, err := NewReview(&productID, nil, 5, "  ", "A", nil, nil, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("empty reviewer name fails validation", func(t *testing.T) {
		_, err := NewReview(&productID, nil, 5, "Good product", "", nil, nil, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("blocked content in comment is auto-hidden, not rejected", func(t *testing.T) {
		r, err := NewReview(&productID, nil, 1, "sản phẩm rác, đồ khốn", "A", nil, nil, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if r.IsPublished() {
			t.Error("expected review with blocked content to be auto-hidden")
		}
	})

	t.Run("blocked content in reviewer name is auto-hidden", func(t *testing.T) {
		r, err := NewReview(&productID, nil, 5, "Sản phẩm tốt", "vay tiền nhanh 0901234567", nil, nil, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if r.IsPublished() {
			t.Error("expected review with blocked reviewer name to be auto-hidden")
		}
	})
}

func TestReview_SetPublished(t *testing.T) {
	productID := "product-1"
	r, _ := NewReview(&productID, nil, 5, "Great", "A", nil, nil, nil)

	r.SetPublished(false)
	if r.IsPublished() {
		t.Error("expected IsPublished to be false after SetPublished(false)")
	}

	r.SetPublished(true)
	if !r.IsPublished() {
		t.Error("expected IsPublished to be true after SetPublished(true)")
	}
}
