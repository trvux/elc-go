package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trvux/elc-go/internal/upload/domain"
)

// fakeS3 speaks just enough of the S3 path-style API (PUT, HEAD, DELETE) for the real AWS
// SDK inside R2FileStore to run against it, so what is tested is our code plus the SDK's own
// request and signing behaviour, not a hand-written stand-in for the store interface.
type fakeS3 struct {
	mu      sync.Mutex
	objects map[string]struct {
		body        []byte
		contentType string
	}
	requests []string
	denyPut  bool
}

func newFakeS3() *fakeS3 {
	return &fakeS3{objects: map[string]struct {
		body        []byte
		contentType string
	}{}}
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	// Path style: /<bucket>/<key>
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	if len(parts) != 2 || parts[0] != "elc-private" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	key := parts[1]
	switch r.Method {
	case http.MethodPut:
		if f.denyPut {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		body, _ := io.ReadAll(r.Body)
		f.objects[key] = struct {
			body        []byte
			contentType string
		}{body, r.Header.Get("Content-Type")}
		w.WriteHeader(http.StatusOK)
	case http.MethodHead:
		o, ok := f.objects[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(o.body)))
		w.Header().Set("Content-Type", o.contentType)
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		delete(f.objects, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func newStoreAgainst(f *fakeS3) (*R2FileStore, func()) {
	srv := httptest.NewServer(f)
	return NewR2FileStore("acct", "AKIATEST", "secret", "elc-private", srv.URL), srv.Close
}

func TestR2FileStore_PutStatDelete(t *testing.T) {
	f := newFakeS3()
	store, stop := newStoreAgainst(f)
	defer stop()
	ctx := context.Background()
	key := "inquiries/202611/" + strings.Repeat("ab", 16) + ".pdf"
	data := []byte("%PDF-1.7 hello")

	if err := store.Put(ctx, key, "application/pdf", bytes.NewReader(data)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if got := f.objects[key]; !bytes.Equal(got.body, data) || got.contentType != "application/pdf" {
		t.Fatalf("stored %q as %q", got.body, got.contentType)
	}
	info, err := store.Stat(ctx, key)
	if err != nil || info.Size != int64(len(data)) || info.ContentType != "application/pdf" {
		t.Fatalf("Stat = %#v, %v", info, err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Stat(ctx, key); !errors.Is(err, domain.ErrFileNotFound) {
		t.Fatalf("Stat after delete = %v, want ErrFileNotFound", err)
	}
}

func TestR2FileStore_StatOfMissingObjectIsErrFileNotFound(t *testing.T) {
	store, stop := newStoreAgainst(newFakeS3())
	defer stop()
	_, err := store.Stat(context.Background(), "inquiries/202611/"+strings.Repeat("00", 16)+".pdf")
	if !errors.Is(err, domain.ErrFileNotFound) {
		t.Fatalf("got %v, want ErrFileNotFound (the admin turns it into 'đã hết hạn')", err)
	}
}

func TestR2FileStore_PutFailureIsReported(t *testing.T) {
	f := newFakeS3()
	f.denyPut = true
	store, stop := newStoreAgainst(f)
	defer stop()
	if err := store.Put(context.Background(), "k", "text/plain", strings.NewReader("x")); err == nil {
		t.Fatal("a refused write must be an error")
	}
}

func TestR2FileStore_SignedURL(t *testing.T) {
	store, stop := newStoreAgainst(newFakeS3())
	defer stop()
	key := "inquiries/202611/" + strings.Repeat("ab", 16) + ".pdf"

	raw, err := store.SignedURL(context.Background(), key, 5*time.Minute, "bản vẽ tầng 1.pdf", "application/pdf", true)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if !strings.HasSuffix(u.Path, "/elc-private/"+key) {
		t.Errorf("path %q does not point at the object", u.Path)
	}
	if q.Get("X-Amz-Expires") != "300" || q.Get("X-Amz-Signature") == "" {
		t.Errorf("link is not a 5-minute signed link: %s", raw)
	}
	if disp := q.Get("response-content-disposition"); !strings.HasPrefix(disp, "inline") || !strings.Contains(disp, "filename") {
		t.Errorf("disposition = %q", disp)
	}
	if q.Get("response-content-type") != "application/pdf" {
		t.Errorf("content type = %q", q.Get("response-content-type"))
	}
	if strings.Contains(raw, "secret") {
		t.Error("the secret key must never appear in a link")
	}

	dl, _ := store.SignedURL(context.Background(), key, time.Minute, "x.dwg", "image/vnd.dwg", false)
	if d, _ := url.Parse(dl); !strings.HasPrefix(d.Query().Get("response-content-disposition"), "attachment") {
		t.Errorf("a drawing should be offered as a download: %s", dl)
	}
}

func TestR2FileStore_SignedURLCannotBeInjectedThroughTheFileName(t *testing.T) {
	store, stop := newStoreAgainst(newFakeS3())
	defer stop()
	raw, _ := store.SignedURL(context.Background(), "k", time.Minute, "a\"; filename=evil.exe; x=\r\nSet-Cookie: a=b", "application/pdf", false)
	u, _ := url.Parse(raw)
	disp := u.Query().Get("response-content-disposition")
	if strings.ContainsAny(disp, "\r\n") {
		t.Errorf("a file name must not smuggle a header line: %q", disp)
	}
}

func TestSelfTestFileStore(t *testing.T) {
	t.Run("passes against a working store and leaves nothing behind", func(t *testing.T) {
		f := newFakeS3()
		store, stop := newStoreAgainst(f)
		defer stop()
		if err := SelfTestFileStore(context.Background(), store); err != nil {
			t.Fatalf("self-test failed: %v", err)
		}
		if len(f.objects) != 0 {
			t.Errorf("the probe object was left in the bucket: %v", f.objects)
		}
	})
	t.Run("fails loudly when writes are refused", func(t *testing.T) {
		f := newFakeS3()
		f.denyPut = true
		store, stop := newStoreAgainst(f)
		defer stop()
		err := SelfTestFileStore(context.Background(), store)
		if err == nil || !strings.Contains(err.Error(), "write") {
			t.Fatalf("got %v, want a write error", err)
		}
	})
	t.Run("fails when the store is not configured", func(t *testing.T) {
		if err := SelfTestFileStore(context.Background(), NewNoopFileStore()); err == nil {
			t.Fatal("an unconfigured store must fail the self-test")
		}
	})
}
