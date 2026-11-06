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

func TestCreateInquiry_ClickChannelAllowsBlankIdentity(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	inquiry, err := CreateInquiry(ctx, repo, domain.CreateInquiryInput{
		Channel: domain.ChannelZalo,
	})
	if err != nil {
		t.Fatalf("expected no error for a click-origin lead with no name/phone yet, got %v", err)
	}
	if inquiry.Name() != "" || inquiry.Phone() != "" {
		t.Errorf("expected blank name/phone to be preserved as-is, got name=%q phone=%q", inquiry.Name(), inquiry.Phone())
	}
	if inquiry.Channel() != domain.ChannelZalo {
		t.Errorf("expected channel zalo, got %s", inquiry.Channel())
	}
}

func TestCreateInquiry_FormChannelStillRequiresIdentity(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	_, err := CreateInquiry(ctx, repo, domain.CreateInquiryInput{
		Channel: domain.ChannelForm,
	})
	if err == nil {
		t.Fatal("expected validation error when channel is form but name/phone are blank")
	}
}

func TestCreateInquiry_RejectsInvalidChannel(t *testing.T) {
	repo := newFakeInquiryRepository()
	ctx := context.Background()

	_, err := CreateInquiry(ctx, repo, domain.CreateInquiryInput{
		Name:    "Nguyen Van A",
		Phone:   "0901234567",
		Channel: "carrier-pigeon",
	})
	if err == nil {
		t.Fatal("expected validation error for an unknown channel")
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

// The public endpoint cannot trust the form's own checks, so the same rules are enforced here.
func createWith(t *testing.T, mutate func(*domain.CreateInquiryInput)) error {
	t.Helper()
	input := domain.CreateInquiryInput{Name: "Nguyen Van A", Phone: "0901234567"}
	mutate(&input)
	_, err := CreateInquiry(context.Background(), newFakeInquiryRepository(), input)
	return err
}

func TestCreateInquiry_NameRules(t *testing.T) {
	for _, name := range []string{"", " ", "   ", "A", " A "} {
		if err := createWith(t, func(i *domain.CreateInquiryInput) { i.Name = name }); err == nil {
			t.Errorf("expected name %q to be rejected", name)
		}
	}
	if err := createWith(t, func(i *domain.CreateInquiryInput) { i.Name = "  Lê An " }); err != nil {
		t.Errorf("a two-letter name with padding should be accepted, got %v", err)
	}
}

func TestCreateInquiry_PhoneMustBeVietnameseMobileOnTheForm(t *testing.T) {
	good := []string{"0901234567", "090 123 4567", "090.123.4567", "090-123-4567", "+84901234567", "84901234567", "0351234567"}
	for _, phone := range good {
		if err := createWith(t, func(i *domain.CreateInquiryInput) { i.Phone = phone }); err != nil {
			t.Errorf("expected phone %q to be accepted, got %v", phone, err)
		}
	}
	bad := []string{"", "123", "0201234567", "090123456", "09012345678", "abcdefghij", "0101234567"}
	for _, phone := range bad {
		if err := createWith(t, func(i *domain.CreateInquiryInput) { i.Phone = phone }); err == nil {
			t.Errorf("expected phone %q to be rejected", phone)
		}
	}
}

func TestCreateInquiry_ClickLeadHasNoPhoneRequirement(t *testing.T) {
	// A Zalo/Messenger/Hotline click has no name or phone yet; staff fill them in later.
	_, err := CreateInquiry(context.Background(), newFakeInquiryRepository(), domain.CreateInquiryInput{
		Channel: domain.ChannelZalo,
	})
	if err != nil {
		t.Fatalf("a click-origin lead must still be accepted without identity, got %v", err)
	}
}

func TestCreateInquiry_MessageAndQualifyDataAreBounded(t *testing.T) {
	long := make([]rune, 2001)
	for i := range long {
		long[i] = 'x'
	}
	msg := string(long)
	if err := createWith(t, func(i *domain.CreateInquiryInput) { i.Message = &msg }); err == nil {
		t.Error("expected a 2001-character message to be rejected")
	}
	ok := string(long[:2000])
	if err := createWith(t, func(i *domain.CreateInquiryInput) { i.Message = &ok }); err != nil {
		t.Errorf("a 2000-character message should be accepted, got %v", err)
	}
	big := `{"k":"` + string(make([]byte, 0)) + stringsRepeat("y", 9000) + `"}`
	if err := createWith(t, func(i *domain.CreateInquiryInput) { i.QualifyData = []byte(big) }); err == nil {
		t.Error("expected oversized qualify data to be rejected")
	}
}

func TestCreateInquiry_AttachmentsMustBeHTTPSURLs(t *testing.T) {
	for _, a := range []string{"javascript:alert(1)", "data:image/png;base64,AAAA", "http://x.example/a.png", "/relative.png", "not a url", " "} {
		if err := createWith(t, func(i *domain.CreateInquiryInput) { i.Attachments = []domain.Attachment{{URL: a}} }); err == nil {
			t.Errorf("expected attachment %q to be rejected", a)
		}
	}
	good := []domain.Attachment{{URL: "https://pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev/inquiries/a.webp"}}
	if err := createWith(t, func(i *domain.CreateInquiryInput) { i.Attachments = good }); err != nil {
		t.Errorf("an https attachment should be accepted, got %v", err)
	}
	seven := []domain.Attachment{{URL: "https://a.example/1"}, {URL: "https://a.example/2"}, {URL: "https://a.example/3"}, {URL: "https://a.example/4"}, {URL: "https://a.example/5"}, {URL: "https://a.example/6"}, {URL: "https://a.example/7"}}
	if err := createWith(t, func(i *domain.CreateInquiryInput) { i.Attachments = seven }); err == nil {
		t.Error("expected more than 6 attachments to be rejected")
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
