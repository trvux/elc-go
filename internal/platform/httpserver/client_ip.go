package httpserver

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP prefers X-Forwarded-For (set by the Nginx reverse proxy in front
// of this service per ARCHITECTURE.md §12) since r.RemoteAddr would
// otherwise always be Nginx's own address. Shared by every module that rate
// limits public endpoints keyed by IP (auth, inquiry, event, ...).
func ClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
