package httpserver

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// New builds the shared chi router with the middleware every module's routes
// are mounted onto in cmd/server/main.go. logger is injected explicitly
// (composition root owns it) rather than accessed as a package global.
//
// extra middleware (the metrics recorder) is installed after accessLog and
// BEFORE recoverer, i.e. outside it, so a panic turned into a 500 by
// recoverer is still seen — and counted — as a 500.
func New(logger *zap.Logger, extra ...func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// accessLog sits OUTSIDE recoverer on purpose: recoverer turns a panic
	// into a normal 500 response, so accessLog then records that 500 instead
	// of never seeing the request finish.
	r.Use(accessLog(logger))
	for _, mw := range extra {
		r.Use(mw)
	}
	r.Use(recoverer(logger))
	return r
}

// recoverer catches panics from any handler, logs the full detail via zap
// (including the request ID so it can be correlated with other log lines
// for the same request), and converts the panic into the same JSON error
// shape as any other apperr — the client never sees a raw stack trace.
func recoverer(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						zap.Any("panic", rec),
						zap.String("request_id", middleware.GetReqID(req.Context())),
					)
					WriteError(w, apperr.NewInternalError(fmt.Errorf("panic: %v", rec)))
				}
			}()
			next.ServeHTTP(w, req)
		})
	}
}
