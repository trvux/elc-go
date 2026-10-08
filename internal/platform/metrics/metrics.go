// Package metrics owns the Prometheus registry and the HTTP/DB instrumentation
// exposed on the service's private metrics listener (see cmd/server/main.go).
//
// Everything here follows one rule: label values must come from a small,
// bounded set. A label fed from the request (raw path, user ID, query) creates
// one time series per distinct value, and an attacker or a crawler can make
// that unbounded — Prometheus then runs out of memory. So the route label is
// the chi route PATTERN ("/products/{id}"), the method is normalized to a
// fixed list, and the status is the numeric code.
package metrics

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// unmatchedRoute is the route label for requests no chi route matched (404s,
// scanners probing random paths). They all share one series on purpose.
const unmatchedRoute = "unmatched"

// Metrics bundles the registry with the HTTP collectors. Use a private
// registry (not prometheus.DefaultRegisterer) so tests can build many
// instances and nothing else can register into the production endpoint.
type Metrics struct {
	Registry *prometheus.Registry

	skip     func(path string) bool
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	respSize *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

// New builds the registry with the Go runtime, process and build-info
// collectors plus the HTTP collectors. skip (may be nil) excludes paths such
// as the health probes from the HTTP series.
func New(skip func(path string) bool) *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		Registry: reg,
		skip:     skip,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests handled, by method, route pattern and status code.",
		}, []string{"method", "route", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency, by method and route pattern.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "route"}),
		respSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_response_size_bytes",
			Help: "HTTP response body size, by route pattern.",
			// 512B .. 32MB. Several list endpoints return 0.1-2MB per call.
			Buckets: prometheus.ExponentialBuckets(512, 4, 9),
		}, []string{"route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_in_flight_requests",
			Help: "HTTP requests currently being served.",
		}),
	}
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewBuildInfoCollector(),
		m.requests, m.duration, m.respSize, m.inFlight,
	)
	return m
}

// Handler serves the registry in the Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{Registry: m.Registry})
}

// Middleware records one observation per finished request. It must sit OUTSIDE
// the panic recoverer (httpserver.New installs it there) so a recovered panic
// is counted as the 500 it became.
func (m *Metrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.skip != nil && m.skip(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			m.inFlight.Inc()
			defer m.inFlight.Dec()

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK // handler wrote nothing: implicit 200
			}
			method := normalizeMethod(r.Method)
			route := routePattern(r)

			m.requests.WithLabelValues(method, route, statusLabel(status)).Inc()
			m.duration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
			m.respSize.WithLabelValues(route).Observe(float64(ww.BytesWritten()))
		})
	}
}

// routePattern must be read AFTER routing has run: chi fills the pattern into
// the same RouteContext the request carries.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	return unmatchedRoute
}

// normalizeMethod collapses anything outside the standard methods into
// "OTHER". net/http accepts arbitrary method tokens, so a scanner sending
// "FOO1", "FOO2", ... would otherwise mint a new series per request.
func normalizeMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodHead, http.MethodOptions:
		return method
	default:
		return "OTHER"
	}
}

func statusLabel(code int) string {
	switch {
	case code >= 100 && code <= 599:
		return itoa(code)
	default:
		return "0"
	}
}

func itoa(n int) string {
	// 3-digit codes only (guarded above); avoids strconv for a hot path.
	return string([]byte{byte('0' + n/100), byte('0' + n/10%10), byte('0' + n%10)})
}
