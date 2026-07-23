package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/shippingzone/domain"
)

func TestLookupZone_WardMatch(t *testing.T) {
	repo := newFakeShippingZoneRepository()
	ctx := context.Background()

	catchAll, _ := CreateZone(ctx, repo, domain.CreateZoneInput{
		Name: "HCM (còn lại)", MinDays: 1, MaxDays: 2, ProvinceCodes: []string{"thanh-pho-ho-chi-minh"},
	})
	innerCity, _ := CreateZone(ctx, repo, domain.CreateZoneInput{
		Name: "Nội thành HCM", MinDays: 0, MaxDays: 1,
		ProvinceCodes: []string{"thanh-pho-ho-chi-minh"},
		WardCodes:     []string{"26884", "27073"}, // Phường Gò Vấp, Phường Phú Nhuận
	})

	got, err := LookupZone(ctx, repo, "thanh-pho-ho-chi-minh", "26884")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || got.ID() != innerCity.ID() {
		t.Fatalf("expected ward-matched zone %s, got %+v (catchAll=%s)", innerCity.ID(), got, catchAll.ID())
	}
}

func TestLookupZone_ProvinceCatchAll(t *testing.T) {
	repo := newFakeShippingZoneRepository()
	ctx := context.Background()

	catchAll, _ := CreateZone(ctx, repo, domain.CreateZoneInput{
		Name: "HCM (còn lại)", MinDays: 1, MaxDays: 2, ProvinceCodes: []string{"thanh-pho-ho-chi-minh"},
	})
	_, _ = CreateZone(ctx, repo, domain.CreateZoneInput{
		Name: "Nội thành HCM", MinDays: 0, MaxDays: 1,
		ProvinceCodes: []string{"thanh-pho-ho-chi-minh"},
		WardCodes:     []string{"26884", "27073"},
	})

	// A ward in the same province, but not part of any ward-narrowed zone.
	got, err := LookupZone(ctx, repo, "thanh-pho-ho-chi-minh", "99999")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || got.ID() != catchAll.ID() {
		t.Fatalf("expected catch-all zone %s, got %+v", catchAll.ID(), got)
	}
}

func TestLookupZone_UnknownProvinceFallsBackToGlobalDefault(t *testing.T) {
	repo := newFakeShippingZoneRepository()
	ctx := context.Background()

	def, _ := CreateZone(ctx, repo, domain.CreateZoneInput{
		Name: "Toàn quốc (mặc định)", MinDays: 3, MaxDays: 7, IsDefault: true,
	})

	got, err := LookupZone(ctx, repo, "an-giang", "12345")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || got.ID() != def.ID() {
		t.Fatalf("expected global default zone %s, got %+v", def.ID(), got)
	}
}

func TestLookupZone_NoZonesConfiguredAtAll(t *testing.T) {
	repo := newFakeShippingZoneRepository()
	ctx := context.Background()

	got, err := LookupZone(ctx, repo, "an-giang", "12345")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil zone when nothing configured, got %+v", got)
	}
}
