package metrics

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func newRouter(m *Metrics) *chi.Mux {
	r := chi.NewRouter()
	// Mirrors httpserver.New: metrics sits OUTSIDE the recoverer.
	r.Use(m.Middleware())
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, req)
		})
	})
	return r
}

func do(r http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func counter(t *testing.T, m *Metrics, method, route string, code int) float64 {
	t.Helper()
	return testutil.ToFloat64(m.requests.WithLabelValues(method, route, strconv.Itoa(code)))
}

func TestMiddleware_LabelsByRoutePatternNotRawPath(t *testing.T) {
	m := New(nil)
	r := newRouter(m)
	r.Get("/products/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("abcd"))
	})

	// Three different ids must land in ONE series.
	for _, id := range []string{"1", "2", "SECRET-TOKEN-3"} {
		do(r, http.MethodGet, "/products/"+id+"?email=a@b.com")
	}

	if got := counter(t, m, "GET", "/products/{id}", 201); got != 3 {
		t.Fatalf("requests{route=/products/{id},code=201} = %v, want 3", got)
	}
	if n := testutil.CollectAndCount(m.requests); n != 1 {
		t.Fatalf("http_requests_total has %d series, want 1 (cardinality leak)", n)
	}
	// Nothing from the URL may appear in the exposition output.
	body := do(m.Handler(), http.MethodGet, "/metrics").Body.String()
	for _, leaked := range []string{"SECRET-TOKEN-3", "a@b.com"} {
		if strings.Contains(body, leaked) {
			t.Errorf("metrics output leaked %q", leaked)
		}
	}
}

func TestMiddleware_UnmatchedAndOddMethodsShareOneSeries(t *testing.T) {
	m := New(nil)
	r := newRouter(m)
	// chi only builds the middleware chain once at least one route exists.
	r.Get("/known", func(http.ResponseWriter, *http.Request) {})

	for _, p := range []string{"/wp-login.php", "/.env", "/a/b/c"} {
		do(r, http.MethodGet, p)
	}
	for _, method := range []string{"FOO1", "FOO2", "PROPFIND"} {
		do(r, method, "/x")
	}

	if got := counter(t, m, "GET", "unmatched", 404); got != 3 {
		t.Errorf("unmatched GET 404 = %v, want 3", got)
	}
	// chi answers any method outside its known set with 405, whatever the path.
	if got := counter(t, m, "OTHER", "unmatched", 405); got != 3 {
		t.Errorf("OTHER-method 405 = %v, want 3 (scanner methods must collapse)", got)
	}
}

func TestMiddleware_PanicIsCountedAs500(t *testing.T) {
	m := New(nil)
	r := newRouter(m)
	r.Get("/boom", func(http.ResponseWriter, *http.Request) { panic("kaboom") })

	do(r, http.MethodGet, "/boom")

	if got := counter(t, m, "GET", "/boom", 500); got != 1 {
		t.Fatalf("panicking request counted %v times as 500, want 1", got)
	}
	if got := testutil.ToFloat64(m.inFlight); got != 0 {
		t.Fatalf("in-flight gauge = %v after the request ended, want 0", got)
	}
}

func TestMiddleware_ImplicitOKAndSkippedPaths(t *testing.T) {
	m := New(func(p string) bool { return p == "/healthz" })
	r := newRouter(m)
	r.Get("/silent", func(http.ResponseWriter, *http.Request) {})
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	do(r, http.MethodGet, "/silent")
	do(r, http.MethodGet, "/healthz")

	if got := counter(t, m, "GET", "/silent", 200); got != 1 {
		t.Errorf("handler that wrote nothing: 200 count = %v, want 1", got)
	}
	if n := testutil.CollectAndCount(m.requests); n != 1 {
		t.Errorf("series = %d, want 1: the probe path must not be recorded", n)
	}
}

func TestMiddleware_RecordsLatencyAndSize(t *testing.T) {
	m := New(nil)
	r := newRouter(m)
	r.Get("/big", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 3000)))
	})
	do(r, http.MethodGet, "/big")

	if n := testutil.CollectAndCount(m.duration); n != 1 {
		t.Errorf("duration histogram series = %d, want 1", n)
	}
	if n := testutil.CollectAndCount(m.respSize); n != 1 {
		t.Errorf("size histogram series = %d, want 1", n)
	}
}

func TestMiddleware_PreservesFlusherForSSE(t *testing.T) {
	m := New(nil)
	r := newRouter(m)
	var flushable bool
	r.Get("/stream", func(w http.ResponseWriter, _ *http.Request) {
		_, flushable = w.(http.Flusher)
	})
	do(r, http.MethodGet, "/stream")
	if !flushable {
		t.Fatal("http.Flusher lost through the metrics wrapper; AI chat SSE would buffer")
	}
}

func TestHandler_ExposesGoProcessAndBuildCollectors(t *testing.T) {
	m := New(nil)
	body := do(m.Handler(), http.MethodGet, "/metrics").Body.String()
	for _, want := range []string{"go_goroutines", "process_start_time_seconds", "go_build_info", "http_in_flight_requests"} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}

func TestStatusLabelAndMethodNormalization(t *testing.T) {
	cases := map[int]string{200: "200", 404: "404", 503: "503", 0: "0", 99: "0", 600: "0"}
	for in, want := range cases {
		if got := statusLabel(in); got != want {
			t.Errorf("statusLabel(%d) = %q, want %q", in, got, want)
		}
	}
	if normalizeMethod("get") != "OTHER" || normalizeMethod("GET") != "GET" {
		t.Error("method normalization must be exact-match on the standard upper-case methods")
	}
}
