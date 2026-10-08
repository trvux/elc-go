package httpserver

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns the visitor's address for per-IP rate limiting and for the
// source_ip stored on inquiries/reviews.
//
// Production path: Cloudflare -> nginx -> Next.js -> this service. nginx
// resolves the real visitor (CF-Connecting-IP, accepted only from Cloudflare's
// ranges) and OVERWRITES X-Forwarded-For with that single address; Next.js
// forwards it on its server-side calls; this service sees only Next.js, via
// the Docker gateway. Before this was wired (RFC 2026-10-08, G7) no hop passed
// the address on, every request looked like the gateway, and every per-IP
// limiter was one shared bucket for all visitors.
//
// X-Forwarded-For is honored ONLY when the direct peer is loopback or a
// private address (Next.js on the same host, or the Docker gateway). The
// service is published on 127.0.0.1 only, so a public peer should never
// appear; if one does, the header is attacker-controlled and is ignored. The
// header is also ignored unless its value parses as an IP, so garbage can't
// reach the VARCHAR(64) source_ip columns or become a limiter key.
//
// Reads the LAST entry, not the first: with one trusted proxy chain,
// "trust N hops, read the Nth from the right" means anything before the last
// entry is client-suppliable text. nginx overwrites the header, so there is
// exactly one entry and first == last; reading the last is the safe choice if
// that ever changes to appending. See
// docs/rfc/2026-09-02-backend-code-review-round2.md §3.7 for the original
// spoofing fix.
func ClientIP(r *http.Request) string {
	peer := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		peer = host
	}

	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" && isInternalPeer(peer) {
		parts := strings.Split(fwd, ",")
		if ip := net.ParseIP(strings.TrimSpace(parts[len(parts)-1])); ip != nil {
			return ip.String()
		}
	}
	return peer
}

// isInternalPeer reports whether the connecting address is one we can trust to
// have set X-Forwarded-For: loopback, or a private range (Docker bridge
// gateway, RFC 1918). An unparsable peer is not trusted.
func isInternalPeer(peer string) bool {
	ip := net.ParseIP(peer)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}
