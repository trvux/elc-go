package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func fieldMap(e observer.LoggedEntry) map[string]any {
	return e.ContextMap()
}

func serve(t *testing.T, r http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestAccessLog_RecordsRoutePatternNotRawPath(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	r := New(zap.New(core))
	r.Get("/auth/verify/{token}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello"))
	})

	serve(t, r, http.MethodGet, "/auth/verify/SECRET-MAGIC-TOKEN?email=a@b.com")

	if logs.Len() != 1 {
		t.Fatalf("logged %d entries, want 1", logs.Len())
	}
	f := fieldMap(logs.All()[0])
	if f["route"] != "/auth/verify/{token}" {
		t.Errorf("route = %v, want the pattern /auth/verify/{token}", f["route"])
	}
	if f["status"] != int64(http.StatusCreated) {
		t.Errorf("status = %v, want 201", f["status"])
	}
	if f["method"] != http.MethodGet {
		t.Errorf("method = %v, want GET", f["method"])
	}
	if f["bytes"] != int64(5) {
		t.Errorf("bytes = %v, want 5", f["bytes"])
	}
	if _, ok := f["duration"]; !ok {
		t.Error("duration field missing")
	}
	// Secrets in the URL must never reach the log, under any field.
	for k, v := range f {
		s, _ := v.(string)
		if strings.Contains(s, "SECRET-MAGIC-TOKEN") || strings.Contains(s, "a@b.com") {
			t.Errorf("field %q leaked URL content: %q", k, s)
		}
	}
}

func TestAccessLog_ImplicitOKAndUnmatchedRoute(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	r := New(zap.New(core))
	r.Get("/silent", func(http.ResponseWriter, *http.Request) {}) // writes nothing

	serve(t, r, http.MethodGet, "/silent")
	serve(t, r, http.MethodGet, "/no/such/route")

	entries := logs.All()
	if len(entries) != 2 {
		t.Fatalf("logged %d entries, want 2", len(entries))
	}
	if got := fieldMap(entries[0])["status"]; got != int64(200) {
		t.Errorf("handler that wrote nothing: status = %v, want implicit 200", got)
	}
	f := fieldMap(entries[1])
	if f["route"] != "unmatched" || f["status"] != int64(404) {
		t.Errorf("unknown path: route=%v status=%v, want unmatched/404", f["route"], f["status"])
	}
	if entries[1].Level != zapcore.InfoLevel {
		t.Errorf("404 logged at %v, want Info (not an error)", entries[1].Level)
	}
}

func TestAccessLog_PanicBecomes500AtErrorLevel(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	r := New(zap.New(core))
	r.Get("/boom", func(http.ResponseWriter, *http.Request) { panic("kaboom") })

	rec := serve(t, r, http.MethodGet, "/boom")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("response status = %d, want 500", rec.Code)
	}

	var access *observer.LoggedEntry
	for _, e := range logs.All() {
		if e.Message == "http request" {
			e := e
			access = &e
		}
	}
	if access == nil {
		t.Fatal("no access log entry for the panicking request (accessLog must wrap recoverer)")
	}
	if access.Level != zapcore.ErrorLevel || fieldMap(*access)["status"] != int64(500) {
		t.Errorf("access entry level=%v status=%v, want Error/500", access.Level, fieldMap(*access)["status"])
	}
}

func TestAccessLog_SkipsProbePaths(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	r := New(zap.New(core))
	RegisterHealthRoutes(r, fakePinger{}, zap.NewNop())

	serve(t, r, http.MethodGet, "/healthz")
	serve(t, r, http.MethodGet, "/readyz")

	if logs.Len() != 0 {
		t.Fatalf("probe requests were logged (%d entries); they would drown real traffic", logs.Len())
	}
}

func TestAccessLog_PreservesFlusherForSSE(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	r := New(zap.New(core))

	var flushable bool
	r.Get("/stream", func(w http.ResponseWriter, _ *http.Request) {
		_, flushable = w.(http.Flusher)
	})
	serve(t, r, http.MethodGet, "/stream")

	if !flushable {
		t.Fatal("http.Flusher lost through the access log wrapper; AI chat SSE would buffer until the end")
	}
}
