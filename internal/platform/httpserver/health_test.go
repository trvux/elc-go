package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type fakePinger struct {
	err   error
	block bool
}

func (f fakePinger) Ping(ctx context.Context) error {
	if f.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.err
}

func newHealthRouter(db Pinger, log *zap.Logger) *chi.Mux {
	r := chi.NewRouter()
	RegisterHealthRoutes(r, db, log)
	return r
}

func TestHealthz_OKWithoutTouchingDatabase(t *testing.T) {
	// A pinger that would fail if called proves /healthz never touches it.
	r := newHealthRouter(fakePinger{err: errors.New("db down")}, zap.NewNop())

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestReadyz(t *testing.T) {
	t.Run("ok when database answers", func(t *testing.T) {
		r := newHealthRouter(fakePinger{}, zap.NewNop())
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("503 and no error text leaked when ping fails", func(t *testing.T) {
		core, logs := observer.New(zap.WarnLevel)
		r := newHealthRouter(fakePinger{err: errors.New("password authentication failed for user elc")}, zap.New(core))

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "password") {
			t.Fatalf("response leaked the database error: %s", rec.Body.String())
		}
		if logs.Len() != 1 {
			t.Fatalf("logged %d warnings, want 1 (the failure must be visible in logs)", logs.Len())
		}
	})

	t.Run("503 within the timeout when the database hangs", func(t *testing.T) {
		h := readyz(fakePinger{block: true}, zap.NewNop(), 20*time.Millisecond)

		rec := httptest.NewRecorder()
		start := time.Now()
		h(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Fatalf("took %v, the ping timeout did not bound the handler", elapsed)
		}
	})
}
