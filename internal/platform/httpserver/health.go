package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const (
	healthzPath = "/healthz"
	readyzPath  = "/readyz"

	// readyzTimeout bounds the DB ping so a hung database makes /readyz fail
	// fast (503) instead of tying up the handler until the client gives up.
	readyzTimeout = 2 * time.Second
)

// Pinger is satisfied by *pgxpool.Pool. Declared here — not imported from the
// db package — so this platform package depends on a one-method interface
// instead of a concrete driver, and tests can use a fake.
type Pinger interface {
	Ping(ctx context.Context) error
}

// RegisterHealthRoutes mounts the two probe endpoints on the root router.
//
//   - /healthz is liveness: the process is up and serving HTTP. It touches
//     nothing else, so it is safe to poll every few seconds (Docker
//     HEALTHCHECK) without generating DB load or noisy metrics. The previous
//     probe was GET /brands, a real DB query ~17,000 times a day.
//   - /readyz is readiness: additionally proves the database answers. Meant
//     for deploy gates and dashboards, not for second-by-second polling.
//
// Neither is authenticated, and neither reveals anything beyond ok/unavailable.
// Both are skipped by the access log and the metrics middleware (IsProbePath).
func RegisterHealthRoutes(r chi.Router, db Pinger, log *zap.Logger) {
	r.Get(healthzPath, healthz)
	r.Get(readyzPath, readyz(db, log, readyzTimeout))
}

// IsProbePath reports whether path is a liveness/readiness probe. Probes are
// polled every few seconds, so the access log and the metrics middleware both
// skip them; otherwise they would drown real traffic and distort every rate.
func IsProbePath(path string) bool {
	return path == healthzPath || path == readyzPath
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readyz(db Pinger, log *zap.Logger, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			// The error goes to the log only; the response body stays generic
			// so an unauthenticated caller learns nothing about the database.
			log.Warn("readiness check failed", zap.Error(err))
			WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
