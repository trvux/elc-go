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
//
// Reads the LAST entry of the header, not the first — this deployment has
// exactly one reverse proxy in front of it (Nginx), so exactly one hop of
// X-Forwarded-For is trusted, standard "trust N proxies, read the Nth from
// the right" practice (same idea as Express's `trust proxy` count or
// Django's SECURE_PROXY_SSL_HEADER guidance). Nginx's own
// proxy_add_x_forwarded_for APPENDS the address it directly observed to
// whatever the incoming request already carried, so the last entry is
// always what Nginx itself saw and can't be forged by the client — only
// entries before it can be arbitrary client-supplied text. Reading the
// FIRST entry (the previous behavior) let a client spoof its IP outright to
// dodge per-IP rate limiting (login, ai/chat, inquiry, event) by simply
// prepending a fake address of its choosing. If Nginx instead overwrites
// (rather than appends to) this header, there is only ever one entry, so
// first and last are identical and this change is a no-op for that case.
// See docs/rfc/2026-09-02-backend-code-review-round2.md §3.7.
func ClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
