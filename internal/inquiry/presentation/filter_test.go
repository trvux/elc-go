package presentation

import (
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestParseInquiryFilter_DateRangeIsVietnamDayInclusive(t *testing.T) {
	r := httptest.NewRequest("GET", "/inquiries/export?from=2026-11-01&to=2026-11-03", nil)
	f := parseInquiryFilter(r)

	wantFrom := time.Date(2026, 10, 31, 17, 0, 0, 0, time.UTC) // 00:00 ICT
	wantTo := time.Date(2026, 11, 3, 17, 0, 0, 0, time.UTC)    // 00:00 ICT on Nov 4 (to is inclusive of Nov 3)
	if !f.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", f.From, wantFrom)
	}
	if !f.To.Equal(wantTo) {
		t.Errorf("To = %v, want %v", f.To, wantTo)
	}
}

func TestParseInquiryFilter_BadDatesMeanNoBound(t *testing.T) {
	r := httptest.NewRequest("GET", "/inquiries?from=yesterday&to=2026-13-45", nil)
	f := parseInquiryFilter(r)
	if !f.From.IsZero() || !f.To.IsZero() {
		t.Errorf("malformed dates must be ignored, got From=%v To=%v", f.From, f.To)
	}
}

func TestParseInquiryFilter_IDsKeepOnlyUUIDs(t *testing.T) {
	good := "6f1c2a3e-1b2c-4d5e-8f90-a1b2c3d4e5f6"
	r := httptest.NewRequest("GET", "/inquiries/export?ids="+url.QueryEscape(good+",not-a-uuid,';drop table inquiries;--"), nil)
	f := parseInquiryFilter(r)
	if len(f.IDs) != 1 || f.IDs[0] != good {
		t.Errorf("IDs = %v, want only %s", f.IDs, good)
	}
}

func TestParseInquiryFilter_IDsAreCapped(t *testing.T) {
	id := "6f1c2a3e-1b2c-4d5e-8f90-a1b2c3d4e5f6"
	raw := ""
	for i := 0; i < maxExportIDs+50; i++ {
		raw += id + ","
	}
	f := parseInquiryFilter(httptest.NewRequest("GET", "/inquiries/export?ids="+raw, nil))
	if len(f.IDs) != maxExportIDs {
		t.Errorf("len(IDs) = %d, want %d", len(f.IDs), maxExportIDs)
	}
}
