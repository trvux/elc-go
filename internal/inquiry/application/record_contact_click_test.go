package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

func TestRecordContactClick_CreatesNewLead(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	sessionID := "session-1"
	gclid := "gclid-1"
	inquiry, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{
		Channel:   domain.ChannelZalo,
		SessionID: &sessionID,
		GCLID:     &gclid,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inquiry.Name() != "" || inquiry.Phone() != "" {
		t.Errorf("expected blank name/phone, got name=%q phone=%q", inquiry.Name(), inquiry.Phone())
	}
	if inquiry.Channel() != domain.ChannelZalo {
		t.Errorf("expected channel zalo, got %s", inquiry.Channel())
	}
	if inquiry.Status() != domain.InquiryStatusNew {
		t.Errorf("expected status new, got %s", inquiry.Status())
	}
}

func TestRecordContactClick_DedupsSameSessionAndChannelWhileOpen(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()
	sessionID := "session-1"

	productA := "product-a"
	first, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{
		Channel:   domain.ChannelZalo,
		SessionID: &sessionID,
		ProductID: &productA,
		LeadType:  domain.LeadTypeProduct,
	})
	if err != nil {
		t.Fatalf("first click: expected no error, got %v", err)
	}

	productB := "product-b"
	second, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{
		Channel:   domain.ChannelZalo,
		SessionID: &sessionID,
		ProductID: &productB,
		LeadType:  domain.LeadTypeProduct,
	})
	if err != nil {
		t.Fatalf("second click: expected no error, got %v", err)
	}

	if second.ID() != first.ID() {
		t.Fatalf("expected the second click to refresh the same open row, got a new id (%s vs %s)", second.ID(), first.ID())
	}
	if second.ProductID() == nil || *second.ProductID() != productB {
		t.Errorf("expected refreshed row to carry the latest product touch (%s), got %v", productB, second.ProductID())
	}

	count, err := CountInquiries(ctx, repo, domain.InquiryFilter{})
	if err != nil {
		t.Fatalf("unexpected error counting: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 row after 2 clicks in the same open session+channel, got %d", count)
	}
}

func TestRecordContactClick_DedupKeepsFullInterestHistoryNotJustLatestTouch(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()
	sessionID := "session-1"

	qualifyA := json.RawMessage(`{"entityName":"Máy lạnh Daikin FTKB25ZVMV","pagePath":"/san-pham/a"}`)
	first, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{
		Channel:     domain.ChannelZalo,
		SessionID:   &sessionID,
		QualifyData: qualifyA,
	})
	if err != nil {
		t.Fatalf("first click: expected no error, got %v", err)
	}

	qualifyB := json.RawMessage(`{"entityName":"Vệ sinh bảo trì các dòng máy lạnh","pagePath":"/dich-vu"}`)
	second, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{
		Channel:     domain.ChannelZalo,
		SessionID:   &sessionID,
		QualifyData: qualifyB,
	})
	if err != nil {
		t.Fatalf("second click: expected no error, got %v", err)
	}
	if second.ID() != first.ID() {
		t.Fatalf("expected dedup onto the same row, got a new id (%s vs %s)", second.ID(), first.ID())
	}

	var got struct {
		EntityName      string                 `json:"entityName"`
		InterestHistory []domain.InterestTouch `json:"interestHistory"`
	}
	if err := json.Unmarshal(second.QualifyData(), &got); err != nil {
		t.Fatalf("unexpected error unmarshaling qualify_data: %v", err)
	}
	if got.EntityName != "Vệ sinh bảo trì các dòng máy lạnh" {
		t.Errorf("expected top-level entityName to still be the latest touch, got %q", got.EntityName)
	}
	if len(got.InterestHistory) != 2 {
		t.Fatalf("expected both touches preserved in interestHistory, got %d entries: %+v", len(got.InterestHistory), got.InterestHistory)
	}
	if got.InterestHistory[0].EntityName != "Máy lạnh Daikin FTKB25ZVMV" {
		t.Errorf("expected first entry to be the first click's entity, got %q", got.InterestHistory[0].EntityName)
	}
	if got.InterestHistory[1].EntityName != "Vệ sinh bảo trì các dòng máy lạnh" {
		t.Errorf("expected second entry to be the second click's entity, got %q", got.InterestHistory[1].EntityName)
	}
}

func TestRecordContactClick_DifferentChannelsDoNotDedup(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()
	sessionID := "session-1"

	zalo, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{Channel: domain.ChannelZalo, SessionID: &sessionID})
	if err != nil {
		t.Fatalf("zalo click: expected no error, got %v", err)
	}
	hotline, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{Channel: domain.ChannelHotline, SessionID: &sessionID})
	if err != nil {
		t.Fatalf("hotline click: expected no error, got %v", err)
	}

	if zalo.ID() == hotline.ID() {
		t.Fatal("expected a click on a different channel to create a separate lead, not dedup across channels")
	}
}

func TestRecordContactClick_CreatesNewRowOnceThePreviousOneIsClosed(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()
	sessionID := "session-1"

	first, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{Channel: domain.ChannelZalo, SessionID: &sessionID})
	if err != nil {
		t.Fatalf("first click: expected no error, got %v", err)
	}

	if _, err := UpdateInquiryStatus(ctx, repo, nil, nil, UpdateInquiryStatusInput{ID: first.ID(), Status: domain.InquiryStatusConverted}); err != nil {
		t.Fatalf("unexpected error marking converted: %v", err)
	}

	second, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{Channel: domain.ChannelZalo, SessionID: &sessionID})
	if err != nil {
		t.Fatalf("second click: expected no error, got %v", err)
	}

	if second.ID() == first.ID() {
		t.Fatal("expected a click after the previous lead was converted to start a new lead, not reopen the closed one")
	}
}

func TestRecordContactClick_RejectsInvalidChannel(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	_, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{Channel: "carrier-pigeon"})
	if err == nil {
		t.Fatal("expected validation error for an unknown channel")
	}
}

func TestRecordContactClick_RejectsBlankChannel(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	_, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{})
	if err == nil {
		t.Fatal("expected validation error when channel is blank — unlike the form, there's no sensible default here")
	}
}

func TestRecordContactClick_RejectsInvalidLeadTypeEvenOnRefreshPath(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()
	sessionID := "session-1"

	if _, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{Channel: domain.ChannelZalo, SessionID: &sessionID}); err != nil {
		t.Fatalf("first click: expected no error, got %v", err)
	}

	_, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{
		Channel:   domain.ChannelZalo,
		SessionID: &sessionID,
		LeadType:  "garbage",
	})
	if err == nil {
		t.Fatal("expected validation error for an invalid lead type on the refresh (dedup) path too, not just on first create")
	}
}

func TestRecordContactClick_NoSessionIDAlwaysCreatesNewRow(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	first, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{Channel: domain.ChannelHotline})
	if err != nil {
		t.Fatalf("first click: expected no error, got %v", err)
	}
	second, err := RecordContactClick(ctx, repo, domain.CreateContactClickInput{Channel: domain.ChannelHotline})
	if err != nil {
		t.Fatalf("second click: expected no error, got %v", err)
	}

	if first.ID() == second.ID() {
		t.Fatal("expected 2 rows when no session_id is available to dedup against")
	}
}
