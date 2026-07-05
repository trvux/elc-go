package application

import (
	"context"
	"testing"
	"time"

	"github.com/trvux/elc-go/internal/system-page/domain"
)

func TestSystemPageUseCases(t *testing.T) {
	ctx := context.Background()
	repo := newFakeSystemPageRepository()

	home := domain.RehydrateSystemPage("id-home", "Trang chủ", "home", nil, nil, time.Now(), time.Now())
	news := domain.RehydrateSystemPage("id-news", "Trang tin tức", "tin-tuc", nil, nil, time.Now(), time.Now())
	repo.seed(home)
	repo.seed(news)

	t.Run("GetSystemPages", func(t *testing.T) {
		list, err := GetSystemPages(ctx, repo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("expected 2 items, got %d", len(list))
		}
	})

	t.Run("GetSystemPageBySlug found", func(t *testing.T) {
		p, err := GetSystemPageBySlug(ctx, repo, "tin-tuc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p == nil || p.ID() != "id-news" {
			t.Errorf("expected to find tin-tuc page, got: %+v", p)
		}
	})

	t.Run("GetSystemPageBySlug not found", func(t *testing.T) {
		p, err := GetSystemPageBySlug(ctx, repo, "does-not-exist")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p != nil {
			t.Errorf("expected nil, got: %+v", p)
		}
	})

	t.Run("UpdateSystemPage success", func(t *testing.T) {
		title := "Máy lạnh, Hệ thống khí tươi & Dự án trọn gói"
		desc := "Mô tả trang chủ"
		updated, err := UpdateSystemPage(ctx, repo, domain.UpdateSystemPageInput{
			ID:              "id-home",
			MetaTitle:       &title,
			MetaDescription: &desc,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *updated.MetaTitle() != title || *updated.MetaDescription() != desc {
			t.Errorf("unexpected updated state: %+v", updated)
		}
	})

	t.Run("UpdateSystemPage not found", func(t *testing.T) {
		_, err := UpdateSystemPage(ctx, repo, domain.UpdateSystemPageInput{ID: "does-not-exist"})
		if err == nil {
			t.Fatal("expected not-found error, got nil")
		}
	})

	t.Run("UpdateSystemPage validation error", func(t *testing.T) {
		tooLong := ""
		for i := 0; i < 71; i++ {
			tooLong += "a"
		}
		_, err := UpdateSystemPage(ctx, repo, domain.UpdateSystemPageInput{
			ID:        "id-home",
			MetaTitle: &tooLong,
		})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
	})
}
