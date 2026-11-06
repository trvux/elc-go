package presentation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/trvux/elc-go/internal/inquiry/domain"
	uploaddomain "github.com/trvux/elc-go/internal/upload/domain"
)

// memStore is an in-memory FileStore that records what it was given.
type memStore struct {
	mu      sync.Mutex
	objects map[string]memObject
	putErr  error
}
type memObject struct {
	contentType string
	data        []byte
}

func newMemStore() *memStore { return &memStore{objects: map[string]memObject{}} }

func (s *memStore) Put(_ context.Context, key, contentType string, body io.ReadSeeker) error {
	if s.putErr != nil {
		return s.putErr
	}
	data, _ := io.ReadAll(body)
	s.mu.Lock()
	s.objects[key] = memObject{contentType, data}
	s.mu.Unlock()
	return nil
}
func (s *memStore) Stat(_ context.Context, key string) (uploaddomain.FileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.objects[key]
	if !ok {
		return uploaddomain.FileInfo{}, uploaddomain.ErrFileNotFound
	}
	return uploaddomain.FileInfo{Size: int64(len(o.data)), ContentType: o.contentType}, nil
}
func (s *memStore) SignedURL(_ context.Context, key string, ttl time.Duration, name, contentType string, inline bool) (string, error) {
	disp := "attachment"
	if inline {
		disp = "inline"
	}
	return "https://signed.example/" + key + "?ttl=" + ttl.String() + "&disp=" + disp + "&name=" + name, nil
}
func (s *memStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.objects, key)
	s.mu.Unlock()
	return nil
}

func newTestHandler(store uploaddomain.FileStore, repo domain.InquiryRepository) *InquiryHandler {
	return NewInquiryHandler(repo, nil, store, nil, zap.NewNop())
}

func multipartBody(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	h.Set("Content-Type", "application/octet-stream") // the client's claim is never trusted
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	w.Close()
	return &buf, w.FormDataContentType()
}

func upload(h *InquiryHandler, t *testing.T, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, ct := multipartBody(t, "file", filename, content)
	req := httptest.NewRequest(http.MethodPost, "/inquiries/files", body)
	req.Header.Set("Content-Type", ct)
	req.RemoteAddr = "203.0.113.7:4000"
	rec := httptest.NewRecorder()
	h.UploadFile(rec, req)
	return rec
}

var (
	pdfBytes = append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("x"), 100)...)
	dwgBytes = append([]byte("AC1032\x00\x00\x00\x00\x00\x00"), bytes.Repeat([]byte{1}, 100)...)
	dxfBytes = []byte("  0\nSECTION\n  2\nHEADER\n  0\nENDSEC\n  0\nEOF\n")
	pngBytes = append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{2}, 50)...)
)

func TestUploadFile_AcceptsSupportedTypesByContent(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		content  []byte
		wantType string
		wantExt  string
	}{
		{"pdf", "bản vẽ.pdf", pdfBytes, domain.TypePDF, ".pdf"},
		{"dwg", "tang1.dwg", dwgBytes, domain.TypeDWG, ".dwg"},
		{"dxf", "tang1.dxf", dxfBytes, domain.TypeDXF, ".dxf"},
		{"png", "anh.png", pngBytes, domain.TypePNG, ".png"},
		// The name lies; the bytes decide.
		{"pdf named .jpg", "thuc-ra-la-pdf.jpg", pdfBytes, domain.TypePDF, ".pdf"},
	}
	for _, c := range cases {
		store := newMemStore()
		rec := upload(newTestHandler(store, nil), t, c.filename, c.content)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d: %s", c.name, rec.Code, rec.Body)
			continue
		}
		var resp uploadedFileResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp.Type != c.wantType || !strings.HasSuffix(resp.Key, c.wantExt) || resp.Size != int64(len(c.content)) {
			t.Errorf("%s: response %#v", c.name, resp)
		}
		if !domain.ValidAttachmentKey(resp.Key) {
			t.Errorf("%s: key %q is not a valid attachment key", c.name, resp.Key)
		}
		if got := store.objects[resp.Key]; got.contentType != c.wantType || !bytes.Equal(got.data, c.content) {
			t.Errorf("%s: stored %q (%d bytes)", c.name, got.contentType, len(got.data))
		}
	}
}

func TestUploadFile_RejectsWhatIsNotADrawingOrPhoto(t *testing.T) {
	for name, content := range map[string][]byte{
		"zip":   []byte("PK\x03\x04" + strings.Repeat("z", 100)),
		"exe":   []byte("MZ\x90\x00" + strings.Repeat("e", 100)),
		"html":  []byte("<html><script>alert(1)</script></html>"),
		"empty": {},
		"text":  []byte("just some text"),
		"svg":   []byte(`<svg onload="alert(1)"></svg>`),
	} {
		store := newMemStore()
		rec := upload(newTestHandler(store, nil), t, "tep.pdf", content) // named .pdf on purpose
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, rec.Code)
		}
		if len(store.objects) != 0 {
			t.Errorf("%s: a rejected file must not reach the store", name)
		}
	}
}

func TestUploadFile_SizeLimitIsExactly15MB(t *testing.T) {
	atLimit := append([]byte("%PDF-"), bytes.Repeat([]byte("a"), domain.MaxAttachmentBytes-5)...)
	if rec := upload(newTestHandler(newMemStore(), nil), t, "a.pdf", atLimit); rec.Code != http.StatusOK {
		t.Errorf("a file of exactly 15 MB must be accepted, got %d: %s", rec.Code, rec.Body)
	}
	over := append([]byte("%PDF-"), bytes.Repeat([]byte("a"), domain.MaxAttachmentBytes-4)...)
	store := newMemStore()
	if rec := upload(newTestHandler(store, nil), t, "a.pdf", over); rec.Code != http.StatusBadRequest {
		t.Errorf("a file one byte over 15 MB must be rejected, got %d", rec.Code)
	}
	if len(store.objects) != 0 {
		t.Error("an oversized file must not reach the store")
	}
}

func TestUploadFile_MissingFilePartIsRejected(t *testing.T) {
	body, ct := multipartBody(t, "other", "x.pdf", pdfBytes)
	req := httptest.NewRequest(http.MethodPost, "/inquiries/files", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	newTestHandler(newMemStore(), nil).UploadFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", rec.Code)
	}
}

func TestUploadFile_RateLimitedPerIP(t *testing.T) {
	h := newTestHandler(newMemStore(), nil)
	var last int
	for i := 0; i < 11; i++ {
		last = upload(h, t, "a.pdf", pdfBytes).Code
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("the 11th upload within the window should be 429, got %d", last)
	}
}

func TestUploadFile_RefusesWhenAllSlotsAreBusy(t *testing.T) {
	h := newTestHandler(newMemStore(), nil)
	for i := 0; i < cap(h.fileSlots); i++ {
		h.fileSlots <- struct{}{}
	}
	if code := upload(h, t, "a.pdf", pdfBytes).Code; code != http.StatusTooManyRequests {
		t.Errorf("got %d, want 429 while four uploads are already running", code)
	}
}

func TestUploadFile_StoreFailureIsAServerError(t *testing.T) {
	store := newMemStore()
	store.putErr = errors.New("bucket unreachable")
	if code := upload(newTestHandler(store, nil), t, "a.pdf", pdfBytes).Code; code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", code)
	}
}

func TestVerifyAttachments(t *testing.T) {
	store := newMemStore()
	h := newTestHandler(store, nil)
	rec := upload(h, t, "ban ve.pdf", pdfBytes)
	var up uploadedFileResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &up)
	req := httptest.NewRequest(http.MethodPost, "/inquiries", nil)

	t.Run("replaces the claimed size and type with what the store holds", func(t *testing.T) {
		got, err := h.verifyAttachments(req, []domain.Attachment{{Key: up.Key, Name: "x.pdf", Size: 1, Type: domain.TypePNG}})
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Size != int64(len(pdfBytes)) || got[0].Type != domain.TypePDF {
			t.Errorf("got %#v", got[0])
		}
	})
	t.Run("rejects a key that was never uploaded", func(t *testing.T) {
		ghost := "inquiries/202611/" + strings.Repeat("ab", 16) + ".pdf"
		if _, err := h.verifyAttachments(req, []domain.Attachment{{Key: ghost, Size: 5, Type: domain.TypePDF}}); err == nil {
			t.Error("expected an error for a missing file")
		}
	})
	t.Run("rejects a key that is not one the upload endpoint produces", func(t *testing.T) {
		if _, err := h.verifyAttachments(req, []domain.Attachment{{Key: "healthcheck/x.txt", Size: 5, Type: domain.TypePDF}}); err == nil {
			t.Error("expected an error for a foreign key")
		}
	})
	t.Run("passes legacy public URLs through untouched", func(t *testing.T) {
		in := []domain.Attachment{{URL: "https://pub.example/a.webp"}}
		got, err := h.verifyAttachments(req, in)
		if err != nil || got[0] != in[0] {
			t.Errorf("got %#v, %v", got, err)
		}
	})
}

// repoWithOne serves a single inquiry; every other method would panic if reached.
type repoWithOne struct {
	domain.InquiryRepository
	item *domain.Inquiry
}

func (r repoWithOne) GetByID(context.Context, string) (*domain.Inquiry, error) { return r.item, nil }

func inquiryWith(t *testing.T, attachments []domain.Attachment) *domain.Inquiry {
	t.Helper()
	i, err := domain.NewInquiry("Test", "0912345678", nil, nil, nil, nil, nil, "", nil, nil, attachments, "form",
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func attachmentRequest(h *InquiryHandler, id, index string) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Get("/inquiries/{id}/attachments/{index}", h.AttachmentURL)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/inquiries/"+id+"/attachments/"+index, nil))
	return rec
}

func TestAttachmentURL(t *testing.T) {
	store := newMemStore()
	hUp := newTestHandler(store, nil)
	var pdfFile, dwgFile uploadedFileResponse
	_ = json.Unmarshal(upload(hUp, t, "ban ve.pdf", pdfBytes).Body.Bytes(), &pdfFile)
	_ = json.Unmarshal(upload(hUp, t, "mat bang.dwg", dwgBytes).Body.Bytes(), &dwgFile)
	gone := "inquiries/202511/" + strings.Repeat("cd", 16) + ".pdf"

	inq := inquiryWith(t, []domain.Attachment{
		{Key: pdfFile.Key, Name: pdfFile.Name, Size: pdfFile.Size, Type: pdfFile.Type},
		{Key: dwgFile.Key, Name: dwgFile.Name, Size: dwgFile.Size, Type: dwgFile.Type},
		{Key: gone, Name: "cu.pdf", Size: 10, Type: domain.TypePDF},
		{URL: "https://pub.example/inquiries/old.webp"},
	})
	h := newTestHandler(store, repoWithOne{item: inq})

	t.Run("a PDF opens inline for five minutes", func(t *testing.T) {
		rec := attachmentRequest(h, "any", "0")
		var resp attachmentLinkResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if rec.Code != 200 || resp.ExpiresIn != 300 || !strings.Contains(resp.URL, "disp=inline") || !strings.Contains(resp.URL, "ttl=5m0s") {
			t.Errorf("status %d, %#v", rec.Code, resp)
		}
	})
	t.Run("a drawing is offered as a download", func(t *testing.T) {
		var resp attachmentLinkResponse
		_ = json.Unmarshal(attachmentRequest(h, "any", "1").Body.Bytes(), &resp)
		if !strings.Contains(resp.URL, "disp=attachment") {
			t.Errorf("%#v", resp)
		}
	})
	t.Run("an expired file is a 404", func(t *testing.T) {
		if code := attachmentRequest(h, "any", "2").Code; code != http.StatusNotFound {
			t.Errorf("status %d, want 404", code)
		}
	})
	t.Run("a legacy public photo returns its own URL", func(t *testing.T) {
		var resp attachmentLinkResponse
		_ = json.Unmarshal(attachmentRequest(h, "any", "3").Body.Bytes(), &resp)
		if resp.URL != "https://pub.example/inquiries/old.webp" || resp.ExpiresIn != 0 {
			t.Errorf("%#v", resp)
		}
	})
	t.Run("bad or out-of-range indexes are rejected", func(t *testing.T) {
		for _, idx := range []string{"4", "99", "-1", "abc"} {
			if code := attachmentRequest(h, "any", idx).Code; code != http.StatusNotFound && code != http.StatusBadRequest {
				t.Errorf("index %q: status %d", idx, code)
			}
		}
	})
}

func TestAttachmentResponsesNeverExposeTheKey(t *testing.T) {
	inq := inquiryWith(t, []domain.Attachment{{Key: "inquiries/202611/" + strings.Repeat("ab", 16) + ".pdf", Name: "a.pdf", Size: 5, Type: domain.TypePDF}})
	raw, _ := json.Marshal(toInquiryResponse(inq))
	if strings.Contains(string(raw), "inquiries/202611") || strings.Contains(string(raw), `"key"`) {
		t.Errorf("the list response leaks the storage key: %s", raw)
	}
	if !strings.Contains(string(raw), `"private":true`) {
		t.Errorf("the response should mark the file private: %s", raw)
	}
}
