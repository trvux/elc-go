package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDetectFileType(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		want string
		ext  string
		ok   bool
	}{
		{"pdf", []byte("%PDF-1.7\n%âãÏÓ"), TypePDF, "pdf", true},
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10}, TypeJPEG, "jpg", true},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0}, TypePNG, "png", true},
		{"webp", append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), 0), TypeWebP, "webp", true},
		{"dwg 2018", []byte("AC1032\x00\x00\x00\x00\x00\x00"), TypeDWG, "dwg", true},
		{"dwg r14", []byte("AC1014\x00\x00"), TypeDWG, "dwg", true},
		{"ascii dxf", []byte("  0\r\nSECTION\r\n  2\r\nHEADER\r\n  9\r\n$ACADVER\r\n"), TypeDXF, "dxf", true},
		{"ascii dxf with comment group and BOM", append([]byte{0xEF, 0xBB, 0xBF}, []byte("999\nmade by x\n0\nSECTION\n2\nHEADER\n")...), TypeDXF, "dxf", true},
		{"binary dxf", []byte("AutoCAD Binary DXF\r\n\x1a\x00"), TypeDXF, "dxf", true},

		{"zip renamed", []byte("PK\x03\x04\x14\x00"), "", "", false},
		{"rar", []byte("Rar!\x1a\x07\x00"), "", "", false},
		{"windows exe", []byte("MZ\x90\x00\x03"), "", "", false},
		{"html", []byte("<html><script>alert(1)</script>"), "", "", false},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg">`), "", "", false},
		{"plain text mentioning SECTION", []byte("hello\nSECTION\n"), "", "", false},
		{"binary containing SECTION after a 0 line", append([]byte("0\nSECTION\n"), 0x00, 0x01), "", "", false},
		{"AC but not dwg", []byte("ACME corp"), "", "", false},
		{"empty", nil, "", "", false},
	}
	for _, c := range cases {
		got, ext, ok := DetectFileType(c.head)
		if got != c.want || ext != c.ext || ok != c.ok {
			t.Errorf("%s: got (%q, %q, %v), want (%q, %q, %v)", c.name, got, ext, ok, c.want, c.ext, c.ok)
		}
	}
}

func TestAttachmentJSON_LegacyStringStaysAString(t *testing.T) {
	legacy := Attachment{URL: "https://pub.example/inquiries/a.webp"}
	raw, err := json.Marshal([]Attachment{legacy})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `["https://pub.example/inquiries/a.webp"]` {
		t.Errorf("legacy attachment marshalled as %s, want the bare string it was stored as", raw)
	}
}

func TestAttachmentJSON_ReadsBothShapes(t *testing.T) {
	var got []Attachment
	in := `["https://pub.example/a.webp", {"key":"inquiries/202611/` + strings.Repeat("a", 32) + `.pdf","name":"ban ve.pdf","size":123,"type":"application/pdf"}]`
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].URL != "https://pub.example/a.webp" || got[0].IsPrivate() {
		t.Fatalf("legacy entry read as %#v", got[0])
	}
	if !got[1].IsPrivate() || got[1].Name != "ban ve.pdf" || got[1].Size != 123 {
		t.Fatalf("private entry read as %#v", got[1])
	}
	back, _ := json.Marshal(got)
	var again []Attachment
	if err := json.Unmarshal(back, &again); err != nil || again[1] != got[1] || again[0] != got[0] {
		t.Errorf("round trip changed the data: %s (%v)", back, err)
	}
}

func TestValidAttachmentKey(t *testing.T) {
	good := "inquiries/202611/" + strings.Repeat("0f", 16) + ".pdf"
	if !ValidAttachmentKey(good) {
		t.Errorf("%q should be valid", good)
	}
	for _, bad := range []string{
		"", "../etc/passwd", "inquiries/202611/../x.pdf", "healthcheck/" + strings.Repeat("0f", 16) + ".pdf",
		"inquiries/202611/" + strings.Repeat("0f", 16) + ".exe", "inquiries/202611/short.pdf",
		"inquiries/2026/" + strings.Repeat("0f", 16) + ".pdf", "inquiries/202611/" + strings.Repeat("0F", 16) + ".pdf",
		good + "/extra", "/" + good,
	} {
		if ValidAttachmentKey(bad) {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestSanitizeAttachmentName(t *testing.T) {
	cases := map[string]string{
		`bản vẽ tầng 1.pdf`:      `bản vẽ tầng 1.pdf`,
		`C:\Users\an\ban ve.dwg`: `ban ve.dwg`,
		`../../etc/passwd`:       `passwd`,
		"a\x00b\nc.pdf":          `abc.pdf`,
		"fdp.\u202Egpj":          `fdp.gpj`,
		`   spaced.pdf   `:       `spaced.pdf`,
		strings.Repeat("a", 300): strings.Repeat("a", 120),
		``:                       ``,
	}
	for in, want := range cases {
		if got := SanitizeAttachmentName(in); got != want {
			t.Errorf("SanitizeAttachmentName(%q) = %q, want %q", in, got, want)
		}
	}
}

func privateFile(n int, size int64) Attachment {
	return Attachment{Key: "inquiries/202611/" + strings.Repeat(string(rune('a'+n)), 32) + ".pdf", Name: "f.pdf", Size: size, Type: TypePDF}
}

func TestValidateAttachments_Limits(t *testing.T) {
	if _, errs := validateAttachments([]Attachment{privateFile(0, 15<<20), privateFile(1, 15<<20)}); len(errs) != 0 {
		t.Errorf("two 15 MB files total exactly 30 MB and must be accepted, got %v", errs)
	}
	if _, errs := validateAttachments([]Attachment{privateFile(0, 15<<20), privateFile(1, 15<<20), privateFile(2, 1)}); len(errs) == 0 {
		t.Error("more than 30 MB together must be rejected")
	}
	if _, errs := validateAttachments([]Attachment{privateFile(0, 15<<20+1)}); len(errs) == 0 {
		t.Error("a file over 15 MB must be rejected")
	}
	seven := make([]Attachment, 7)
	for i := range seven {
		seven[i] = privateFile(i, 10)
	}
	if _, errs := validateAttachments(seven); len(errs) == 0 {
		t.Error("seven files must be rejected")
	}
	six := seven[:6]
	if _, errs := validateAttachments(six); len(errs) != 0 {
		t.Errorf("six small files must be accepted, got %v", errs)
	}
}

func TestValidateAttachments_RejectsForgedPrivateEntries(t *testing.T) {
	good := privateFile(0, 100)
	bad := []Attachment{
		{Key: "somewhere/else.pdf", Size: 100, Type: TypePDF},
		{Key: good.Key, Size: 100, Type: "application/zip"},
		{Key: good.Key, Size: 0, Type: TypePDF},
		{Key: good.Key, Size: -5, Type: TypePDF},
		{URL: "javascript:alert(1)"},
		{},
	}
	for i, a := range bad {
		if _, errs := validateAttachments([]Attachment{a}); len(errs) == 0 {
			t.Errorf("forged entry %d (%#v) should be rejected", i, a)
		}
	}
}

func TestValidateAttachments_NormalisesPrivateEntriesAndDropsExtraFields(t *testing.T) {
	in := privateFile(0, 100)
	in.Name = "dir/ban\x00 ve.pdf"
	in.URL = "https://evil.example/x" // a private entry never keeps a URL
	out, errs := validateAttachments([]Attachment{in})
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	if out[0].Name != "ban ve.pdf" || out[0].URL != "" {
		t.Errorf("got %#v", out[0])
	}
}
