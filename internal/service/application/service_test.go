package application

import (
	"context"
	"errors"
	"testing"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/service/domain"
)

func ptrInt64(v int64) *int64 { return &v }
func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }

func TestCreateService(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	s, err := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong",
		OriginalPrice: ptrInt64(100000), DiscountPercent: ptrInt(10),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.ID() == "" {
		t.Error("expected an ID after create")
	}
	if *s.SalePrice() != 90000 {
		t.Errorf("expected sale price 90000, got %d", *s.SalePrice())
	}
}

func TestCreateService_TitleAlignDefaultsToLeft(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	s, err := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.TitleAlign() != domain.TitleAlignLeft {
		t.Errorf("expected titleAlign to default to left, got %s", s.TitleAlign())
	}
}

func TestCreateService_InvalidTitleAlign(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	_, err := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong", TitleAlign: "sideways",
	})
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Fields["titleAlign"]) == 0 {
		t.Errorf("expected titleAlign field error, got %v", appErr.Fields)
	}
}

func TestUpdateService_TitleAlign(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong",
	})

	center := domain.TitleAlignCenter
	updated, err := UpdateService(ctx, repo, domain.UpdateServiceInput{
		ID: created.ID(), TitleAlign: &center,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.TitleAlign() != domain.TitleAlignCenter {
		t.Errorf("expected titleAlign updated to center, got %s", updated.TitleAlign())
	}
}

func TestUpdateService_InvalidTitleAlign(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh treo tuong", Slug: "may-lanh-treo-tuong",
	})

	invalid := "diagonal"
	_, err := UpdateService(ctx, repo, domain.UpdateServiceInput{
		ID: created.ID(), TitleAlign: &invalid,
	})
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Fields["titleAlign"]) == 0 {
		t.Errorf("expected titleAlign field error, got %v", appErr.Fields)
	}
}

func TestUpdateService_NotFound(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	_, err := UpdateService(ctx, repo, domain.UpdateServiceInput{ID: "missing"})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %v", err)
	}
}

func TestUpdateService_PartialPriceUpdate_NeverStale(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh", Slug: "may-lanh",
		OriginalPrice: ptrInt64(200000), DiscountPercent: ptrInt(10),
	})
	if *created.SalePrice() != 180000 {
		t.Fatalf("expected initial sale price 180000, got %d", *created.SalePrice())
	}

	// Update ONLY discountPercent — the exact scenario that went stale before.
	updated, err := UpdateService(ctx, repo, domain.UpdateServiceInput{
		ID: created.ID(), DiscountPercent: ptrInt(50),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if *updated.SalePrice() != 100000 {
		t.Errorf("expected recomputed sale price 100000, got %d", *updated.SalePrice())
	}
	// originalPrice must be untouched.
	if *updated.OriginalPrice() != 200000 {
		t.Errorf("expected originalPrice to remain 200000, got %d", *updated.OriginalPrice())
	}
}

func TestUpdateService_PartialUpdate_UnrelatedFieldsUnchanged(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{
		Title: "May lanh", Slug: "may-lanh", OrderIndex: 3,
	})

	newTitle := "May lanh cao cap"
	updated, err := UpdateService(ctx, repo, domain.UpdateServiceInput{
		ID: created.ID(), Title: ptrStr(newTitle),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Title() != newTitle {
		t.Errorf("expected title updated, got %s", updated.Title())
	}
	if updated.OrderIndex() != 3 {
		t.Errorf("expected orderIndex to remain 3, got %d", updated.OrderIndex())
	}
}

func TestDeleteAndRestoreService(t *testing.T) {
	repo := newFakeServiceRepository()
	ctx := context.Background()

	created, _ := CreateService(ctx, repo, domain.CreateServiceInput{Title: "May lanh", Slug: "may-lanh"})

	if err := DeleteService(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetServiceByID(ctx, repo, created.ID()); found != nil {
		t.Error("expected soft-deleted service to not be found by GetByID")
	}

	if err := RestoreService(ctx, repo, created.ID()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found, _ := GetServiceByID(ctx, repo, created.ID()); found == nil {
		t.Error("expected restored service to be found again")
	}
}
