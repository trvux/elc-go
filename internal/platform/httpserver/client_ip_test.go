package httpserver

import (
	"net/http"
	"strings"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		forwarded  string
		remoteAddr string
		want       string
	}{
		{
			name:       "single hop (Nginx overwrites, doesn't append)",
			forwarded:  "203.0.113.5",
			remoteAddr: "10.0.0.1:12345",
			want:       "203.0.113.5",
		},
		{
			name:       "multiple hops: trusts the LAST one (Nginx's own append), not the client-suppliable first",
			forwarded:  "1.2.3.4, 203.0.113.5",
			remoteAddr: "10.0.0.1:12345",
			want:       "203.0.113.5",
		},
		{
			name:       "spoofed first hop must not win",
			forwarded:  "9.9.9.9",
			remoteAddr: "10.0.0.1:12345",
			want:       "9.9.9.9", // no proxy hop to strip in this single-entry case
		},
		{
			name:       "public peer: header is attacker-controlled, ignored",
			forwarded:  "1.2.3.4",
			remoteAddr: "203.0.113.50:4444",
			want:       "203.0.113.50",
		},
		{
			name:       "loopback peer (Next.js on the same host) is trusted",
			forwarded:  "198.51.100.7",
			remoteAddr: "127.0.0.1:5555",
			want:       "198.51.100.7",
		},
		{
			name:       "Docker gateway peer is trusted",
			forwarded:  "198.51.100.8",
			remoteAddr: "172.18.0.1:6666",
			want:       "198.51.100.8",
		},
		{
			name:       "garbage header falls back to the peer instead of becoming a key",
			forwarded:  "not-an-ip; DROP TABLE",
			remoteAddr: "172.18.0.1:6666",
			want:       "172.18.0.1",
		},
		{
			name:       "oversized header can't reach a VARCHAR(64) column",
			forwarded:  strings.Repeat("9", 200),
			remoteAddr: "172.18.0.1:6666",
			want:       "172.18.0.1",
		},
		{
			name:       "IPv6 visitor is normalized",
			forwarded:  "2001:DB8::1",
			remoteAddr: "172.18.0.1:6666",
			want:       "2001:db8::1",
		},
		{
			name:       "whitespace around the entry is trimmed",
			forwarded:  "  198.51.100.9  ",
			remoteAddr: "172.18.0.1:6666",
			want:       "198.51.100.9",
		},
		{
			name:       "no X-Forwarded-For falls back to RemoteAddr",
			forwarded:  "",
			remoteAddr: "203.0.113.9:54321",
			want:       "203.0.113.9",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}, RemoteAddr: c.remoteAddr}
			if c.forwarded != "" {
				r.Header.Set("X-Forwarded-For", c.forwarded)
			}
			got := ClientIP(r)
			if got != c.want {
				t.Errorf("ClientIP() = %q, want %q", got, c.want)
			}
		})
	}
}
