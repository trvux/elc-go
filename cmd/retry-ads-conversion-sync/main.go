// Command retry-ads-conversion-sync re-attempts the close_convert_lead push
// to GA4/Ads for every converted lead whose first attempt failed (see
// internal/inquiry/application.UpdateInquiryStatus — a failed push there
// never fails the status update itself, it just leaves
// ads_conversion_synced_at nil for this command to find later). A plain
// one-shot CLI like every other cmd/ here — `go run
// ./cmd/retry-ads-conversion-sync` runs it once; run it by hand after
// noticing pending syncs (e.g. in InquiryManagement's "Chưa đồng bộ Google
// Ads" indicator), or wire it into a scheduled workflow later if that
// becomes a recurring need.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	adsconversioninfra "github.com/trvux/elc-go/internal/adsconversion/infrastructure"
	inquiryinfra "github.com/trvux/elc-go/internal/inquiry/infrastructure"
	"github.com/trvux/elc-go/internal/platform/db"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()

	measurementID := os.Getenv("GA4_MEASUREMENT_ID")
	apiSecret := os.Getenv("GA4_API_SECRET")
	if measurementID == "" || apiSecret == "" {
		fatalf("GA4_MEASUREMENT_ID/GA4_API_SECRET must be set — nothing to retry with")
	}
	notifier := adsconversioninfra.NewMeasurementProtocolClient(measurementID, apiSecret)

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	repo := inquiryinfra.NewPostgresInquiryRepository(pool)

	pending, err := repo.FindPendingAdsConversionSync(ctx)
	if err != nil {
		fatalf("find pending: %v", err)
	}
	if len(pending) == 0 {
		fmt.Println("retry-ads-conversion-sync: nothing pending")
		return
	}
	fmt.Printf("retry-ads-conversion-sync: %d lead(s) pending\n", len(pending))

	succeeded, failed := 0, 0
	for _, inquiry := range pending {
		gaClientID := inquiry.GAClientID()
		if gaClientID == nil || *gaClientID == "" {
			// FindPendingAdsConversionSync already filters this at the SQL
			// level — reached only if that query and this check ever drift
			// apart, so fail loud rather than silently skip.
			fmt.Fprintf(os.Stderr, "retry-ads-conversion-sync: %s: no ga_client_id despite query filter, skipping\n", inquiry.ID())
			failed++
			continue
		}

		if err := notifier.SendCloseConvertLead(ctx, *gaClientID, inquiry.ConversionValue()); err != nil {
			fmt.Fprintf(os.Stderr, "retry-ads-conversion-sync: %s: %v\n", inquiry.ID(), err)
			failed++
			continue
		}

		inquiry.MarkAdsConversionSynced()
		if _, err := repo.UpdateAdsConversionSync(ctx, inquiry); err != nil {
			fmt.Fprintf(os.Stderr, "retry-ads-conversion-sync: %s: sent but failed to record synced_at: %v\n", inquiry.ID(), err)
			failed++
			continue
		}

		succeeded++
	}

	fmt.Printf("retry-ads-conversion-sync: done — %d succeeded, %d failed\n", succeeded, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
