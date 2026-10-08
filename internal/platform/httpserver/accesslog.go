package httpserver

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// accessLog writes one structured line per finished request: method, route
// pattern, status, duration, bytes and request_id (the same ID recoverer logs
// on a panic, so the two can be joined).
//
// Deliberately NOT logged: the raw URL path and query string — routes carry
// secrets and PII (magic-link tokens, emails), so only the chi route pattern
// ("/products/{id}") is recorded. Unmatched requests log "unmatched".
//
// client_ip comes from ClientIP (the same value the rate limiters key on). It
// is logged for abuse investigation and to verify the nginx -> Next.js -> Go
// forwarding chain; nginx's own access log already records visitor IPs.
//
// 5xx logs at Error, everything else at Info (a 404 is a normal answer here:
// the Next.js frontend probes for slugs that may not exist).
//
// Probe paths (/healthz, /readyz) are skipped. The response writer is wrapped
// with chi's NewWrapResponseWriter, which preserves http.Flusher/Hijacker, so
// the AI chat's SSE streaming keeps flushing through it.
func accessLog(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isProbePath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				// Handler wrote nothing: net/http sends an implicit 200.
				status = http.StatusOK
			}

			// RoutePattern is only filled in once routing has run, so it must
			// be read AFTER next.ServeHTTP. chi mutates the same RouteContext
			// the request carries, so r.Context() still sees the result.
			route := "unmatched"
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				if pattern := rctx.RoutePattern(); pattern != "" {
					route = pattern
				}
			}

			fields := []zap.Field{
				zap.String("request_id", middleware.GetReqID(r.Context())),
				zap.String("method", r.Method),
				zap.String("route", route),
				zap.String("client_ip", ClientIP(r)),
				zap.Int("status", status),
				zap.Duration("duration", time.Since(start)),
				zap.Int("bytes", ww.BytesWritten()),
			}
			if status >= http.StatusInternalServerError {
				log.Error("http request", fields...)
				return
			}
			log.Info("http request", fields...)
		})
	}
}
